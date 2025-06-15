package application

import (
	"context"
	"fmt"
	"log"
	"pyrolytics/config"
	"pyrolytics/internal/event_ingestor/domain"
	"pyrolytics/pkg/messaging"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

type DEXSubscriber struct {
	wsClient    *ws.Client
	rpcClient   *rpc.Client
	ctx         context.Context
	cancel      context.CancelFunc
	subscribers map[string]*ws.LogSubscription
	natsClient  *messaging.NatsClient
}

func NewDEXSubscriber(cfg *config.Config, natsClient *messaging.NatsClient) *DEXSubscriber {
	ctx, cancel := context.WithCancel(context.Background())

	// Connect to Devnet
	wsClient, err := ws.Connect(ctx, cfg.Solana.WSURL)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket: %v", err)
	}

	rpcClient := rpc.New(cfg.Solana.RPCURL)

	return &DEXSubscriber{
		wsClient:    wsClient,
		rpcClient:   rpcClient,
		ctx:         ctx,
		cancel:      cancel,
		subscribers: make(map[string]*ws.LogSubscription),
		natsClient:  natsClient,
	}
}

func (ds *DEXSubscriber) Close() {
	for _, sub := range ds.subscribers {
		sub.Unsubscribe()
	}
	ds.wsClient.Close()
	ds.cancel()
}

func (ds *DEXSubscriber) SubscribeToProgram(programID solana.PublicKey, programName string) error {
	log.Printf("🔗 Subscribing to %s program: %s", programName, programID.String())

	sub, err := ds.wsClient.LogsSubscribeMentions(
		programID,
		rpc.CommitmentProcessed,
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s logs: %w", programName, err)
	}

	ds.subscribers[programName] = sub

	// Start listening for events in a goroutine
	go ds.listenForEvents(sub, programName, programID)

	return nil
}

// listenForEvents processes incoming log events
func (ds *DEXSubscriber) listenForEvents(sub *ws.LogSubscription, programName string, programID solana.PublicKey) {
	for {
		select {
		case <-ds.ctx.Done():
			return
		default:
			got, err := sub.Recv(ds.ctx)
			if err != nil {
				log.Printf("❌ Error receiving logs for %s: %v", programName, err)
				return
			}

			ds.processLogEvent(got, programName, programID)
		}
	}
}

func (ds *DEXSubscriber) processLogEvent(result *ws.LogResult, programName string, programID solana.PublicKey) {
	logs := result.Value.Logs
	signature := result.Value.Signature.String()
	isSuccess := result.Value.Err == nil

	// Filter for swap-related events
	if ds.isSwapRelated(logs) || ds.isLiquidityRelated(logs) {

		// Create structured event
		event := ds.createSolanaRawEvent(result, programName, programID, logs, isSuccess)

		// Fetch additional transaction details asynchronously
		go ds.enrichEventWithTransactionDetails(event, signature)

		// Display event for debugging
		ds.displayEvent(event)

		// Publish to NATS JetStream
		if err := ds.natsClient.Publish(ds.ctx, event); err != nil {
			log.Printf("❌ Failed to publish event to NATS: %v", err)
		}
	}
}

// createSolanaRawEvent creates a structured event from log data
func (ds *DEXSubscriber) createSolanaRawEvent(result *ws.LogResult, programName string, programID solana.PublicKey, logs []string, isSuccess bool) *domain.SolanaRawEvent {
	signature := result.Value.Signature.String()

	// Generate unique event ID
	eventID := fmt.Sprintf("%s_%d_%d", signature, result.Context.Slot, time.Now().UnixNano())

	// Determine event type
	eventType := ds.determineEventType(logs)

	// Parse DEX-specific data
	parsedData := ds.parseSwapData(logs, programName)

	// Extract swap details if it's a swap event
	var swapDetails *domain.SwapDetails
	if eventType == "swap" {
		swapDetails = ds.extractSwapDetails(logs, parsedData, programName)
	}

	event := &domain.SolanaRawEvent{
		EventID:     eventID,
		EventType:   eventType,
		Blockchain:  "solana",
		Network:     "devnet",
		Timestamp:   time.Now(),
		Signature:   signature,
		Slot:        result.Context.Slot,
		Success:     isSuccess,
		DEXName:     ds.extractDEXName(programName),
		DEXProgram:  ds.extractDEXProgram(programName),
		ProgramID:   programID.String(),
		SwapDetails: swapDetails,
		RawLogs:     logs,
		ParsedData:  parsedData,
	}

	return event
}

// enrichEventWithTransactionDetails fetches and adds transaction details
func (ds *DEXSubscriber) enrichEventWithTransactionDetails(event *domain.SolanaRawEvent, signature string) {
	sig := solana.MustSignatureFromBase58(signature)

	maxSupportedVersion := uint64(0)
	tx, err := ds.rpcClient.GetTransaction(
		context.Background(),
		sig,
		&rpc.GetTransactionOpts{
			Encoding:                       solana.EncodingBase64,
			MaxSupportedTransactionVersion: &maxSupportedVersion,
		},
	)
	if err != nil {
		log.Printf("❌ Failed to fetch transaction %s: %v", signature, err)
		return
	}

	// Enrich event with transaction details
	if tx.BlockTime != nil {
		unixTime := tx.BlockTime.Time().Unix()
		event.BlockTime = &unixTime
	}

	if tx.Meta != nil {
		event.Fee = tx.Meta.Fee

		// Extract token balance changes
		event.TokenBalanceChanges = ds.extractTokenBalanceChanges(tx.Meta)
	}

	// Re-publish updated event
	if err := ds.natsClient.Publish(ds.ctx, event); err != nil {
		log.Printf("❌ Failed to re-publish enriched event to NATS: %v", err)
	}
}

// extractTokenBalanceChanges extracts token balance changes from transaction meta
func (ds *DEXSubscriber) extractTokenBalanceChanges(meta *rpc.TransactionMeta) []domain.TokenBalanceChange {
	var changes []domain.TokenBalanceChange

	// Compare pre and post token balances
	for i, preBalance := range meta.PreTokenBalances {
		if i < len(meta.PostTokenBalances) {
			postBalance := meta.PostTokenBalances[i]

			// Calculate change
			change := "0"
			if preBalance.UiTokenAmount.Amount != postBalance.UiTokenAmount.Amount {
				// Simple string-based calculation (for production, use decimal arithmetic)
				change = fmt.Sprintf("%.6f", *postBalance.UiTokenAmount.UiAmount-*preBalance.UiTokenAmount.UiAmount)
			}

			changes = append(changes, domain.TokenBalanceChange{
				AccountIndex: uint8(preBalance.AccountIndex),
				Mint:         preBalance.Mint.String(),
				Owner:        preBalance.Owner.String(),
				PreBalance:   preBalance.UiTokenAmount.Amount,
				PostBalance:  postBalance.UiTokenAmount.Amount,
				Change:       change,
				Decimals:     preBalance.UiTokenAmount.Decimals,
			})
		}
	}

	return changes
}

// determineEventType determines the type of event based on logs
func (ds *DEXSubscriber) determineEventType(logs []string) string {
	for _, log := range logs {
		logLower := strings.ToLower(log)
		if strings.Contains(logLower, "swap") {
			return "swap"
		}
		if strings.Contains(logLower, "addliquidity") || strings.Contains(logLower, "increase") {
			return "liquidity_add"
		}
		if strings.Contains(logLower, "removeliquidity") || strings.Contains(logLower, "decrease") {
			return "liquidity_remove"
		}
	}
	return "unknown"
}

// extractDEXName extracts DEX name from program name
func (ds *DEXSubscriber) extractDEXName(programName string) string {
	if strings.Contains(programName, "Raydium") {
		return "Raydium"
	}
	if strings.Contains(programName, "Orca") {
		return "Orca"
	}
	return "Unknown"
}

// extractDEXProgram extracts DEX program type from program name
func (ds *DEXSubscriber) extractDEXProgram(programName string) string {
	if strings.Contains(programName, "AMMV4") {
		return "AMM_V4"
	}
	if strings.Contains(programName, "CPMM") {
		return "CPMM"
	}
	if strings.Contains(programName, "CLMM") {
		return "CLMM"
	}
	if strings.Contains(programName, "Whirlpool") {
		return "Whirlpool"
	}
	if strings.Contains(programName, "Routing") {
		return "Routing"
	}
	return "Unknown"
}

// extractSwapDetails extracts swap-specific details from logs and parsed data
func (ds *DEXSubscriber) extractSwapDetails(logs []string, parsedData map[string]interface{}, programName string) *domain.SwapDetails {
	details := &domain.SwapDetails{}

	// Extract information from logs
	for _, log := range logs {
		if strings.Contains(log, "SwapBaseIn") {
			details.Direction = "buy"
			// Extract amount if possible
			if parts := strings.Split(log, ": "); len(parts) > 1 {
				details.AmountIn = parts[1]
			}
		}
		if strings.Contains(log, "SwapBaseOut") {
			details.Direction = "sell"
			if parts := strings.Split(log, ": "); len(parts) > 1 {
				details.AmountOut = parts[1]
			}
		}
	}

	// Extract from parsed data
	if direction, ok := parsedData["swap_direction"].(string); ok {
		details.Direction = direction
	}
	if amountIn, ok := parsedData["amount_in"].(string); ok {
		details.AmountIn = amountIn
	}
	if amountOut, ok := parsedData["amount_out"].(string); ok {
		details.AmountOut = amountOut
	}

	return details
}

// isSwapRelated checks if logs contain swap-related keywords
func (ds *DEXSubscriber) isSwapRelated(logs []string) bool {
	swapKeywords := []string{
		"swap", "Swap", "trade", "Trade", "exchange", "Exchange",
		"buy", "sell", "Buy", "Sell", "SwapBaseIn", "SwapBaseOut",
	}

	return ds.containsKeywords(logs, swapKeywords)
}

// isLiquidityRelated checks if logs contain liquidity-related keywords
func (ds *DEXSubscriber) isLiquidityRelated(logs []string) bool {
	liquidityKeywords := []string{
		"addliquidity", "AddLiquidity", "removeliquidity", "RemoveLiquidity",
		"increaseliquidity", "IncreaseLiquidity", "decreaseliquidity", "DecreaseLiquidity",
		"deposit", "withdraw", "Deposit", "Withdraw",
	}

	return ds.containsKeywords(logs, liquidityKeywords)
}

// containsKeywords checks if logs contain any of the specified keywords
func (ds *DEXSubscriber) containsKeywords(logs []string, keywords []string) bool {
	for _, log := range logs {
		logLower := strings.ToLower(log)
		for _, keyword := range keywords {
			if strings.Contains(logLower, strings.ToLower(keyword)) {
				return true
			}
		}
	}
	return false
}

// parseSwapData extracts swap-specific information from logs (keeping existing logic)
func (ds *DEXSubscriber) parseSwapData(logs []string, programName string) map[string]interface{} {
	data := make(map[string]interface{})

	switch {
	case strings.Contains(programName, "AMMV4"):
		data = ds.parseRaydiumLogs(logs)
	case strings.Contains(programName, "CPMM"):
		data = ds.parseRaydiumCPMMLogs(logs)
	case strings.Contains(programName, "CLMM"):
		data = ds.parseRaydiumCLMMLogs(logs)
	case strings.Contains(programName, "Orca"):
		data = ds.parseOrcaLogs(logs)
	}

	// Extract common information
	for _, log := range logs {
		if strings.Contains(log, "Amount") {
			data["raw_amount_log"] = log
		}
		if strings.Contains(log, "Token") {
			data["raw_token_log"] = log
		}
	}

	return data
}

// parseRaydiumLogs parses Raydium AMM V4 specific logs
func (ds *DEXSubscriber) parseRaydiumLogs(logs []string) map[string]interface{} {
	data := make(map[string]interface{})
	data["dex_type"] = "Raydium AMM V4"

	for _, log := range logs {
		if strings.Contains(log, "Program log: ray_log") {
			data["raydium_log"] = log
		}
		if strings.Contains(log, "swap") {
			data["swap_instruction"] = true
		}
	}

	return data
}

// parseRaydiumCPMMLogs parses Raydium CPMM logs
func (ds *DEXSubscriber) parseRaydiumCPMMLogs(logs []string) map[string]interface{} {
	data := make(map[string]interface{})
	data["dex_type"] = "Raydium CPMM"

	for _, log := range logs {
		if strings.Contains(log, "SwapBaseIn") || strings.Contains(log, "SwapBaseOut") {
			data["swap_direction"] = log
		}
	}

	return data
}

// parseRaydiumCLMMLogs parses Raydium CLMM (Concentrated Liquidity) logs
func (ds *DEXSubscriber) parseRaydiumCLMMLogs(logs []string) map[string]interface{} {
	data := make(map[string]interface{})
	data["dex_type"] = "Raydium CLMM"

	for _, log := range logs {
		if strings.Contains(log, "tick") {
			data["tick_info"] = log
		}
	}

	return data
}

// parseOrcaLogs parses Orca Whirlpool logs
func (ds *DEXSubscriber) parseOrcaLogs(logs []string) map[string]interface{} {
	data := make(map[string]interface{})
	data["dex_type"] = "Orca Whirlpool"

	for _, log := range logs {
		if strings.Contains(log, "whirlpool") {
			data["whirlpool_log"] = log
		}
		if strings.Contains(log, "Swap") {
			data["swap_event"] = log
		}
	}

	return data
}

// displayEvent prints formatted event information
func (ds *DEXSubscriber) displayEvent(event *domain.SolanaRawEvent) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("🔄 %s EVENT on %s (%s)\n", strings.ToUpper(event.EventType), event.DEXName, event.DEXProgram)
	fmt.Printf("📝 Event ID: %s\n", event.EventID)
	fmt.Printf("📝 Signature: %s\n", event.Signature)
	fmt.Printf("📍 Slot: %d\n", event.Slot)
	fmt.Printf("⏰ Time: %s\n", event.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("✅ Success: %v\n", event.Success)

	if event.SwapDetails != nil {
		fmt.Printf("🔄 Swap Details:\n")
		fmt.Printf("   Direction: %s\n", event.SwapDetails.Direction)
		if event.SwapDetails.AmountIn != "" {
			fmt.Printf("   Amount In: %s\n", event.SwapDetails.AmountIn)
		}
		if event.SwapDetails.AmountOut != "" {
			fmt.Printf("   Amount Out: %s\n", event.SwapDetails.AmountOut)
		}
	}

	if len(event.ParsedData) > 0 {
		fmt.Printf("📊 Parsed Data:\n")
		for key, value := range event.ParsedData {
			fmt.Printf("   %s: %v\n", key, value)
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}
