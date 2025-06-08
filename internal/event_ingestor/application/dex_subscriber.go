package application

import (
	"context"
	"fmt"
	"log"
	"pyrolytics/config"
	"pyrolytics/internal/event_ingestor/domain"
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
}

func NewDEXSubscriber(cfg *config.Config) *DEXSubscriber {
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

	// Subscribe to logs mentioning this program
	sub, err := ds.wsClient.LogsSubscribeMentions(
		programID,
		rpc.CommitmentProcessed, // Use processed for faster updates, finalized for more reliable
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

// processLogEvent analyzes and processes individual log events
func (ds *DEXSubscriber) processLogEvent(result *ws.LogResult, programName string, programID solana.PublicKey) {
	if result.Value.Err != nil {
		// Skip failed transactions
		return
	}

	logs := result.Value.Logs
	signature := result.Value.Signature.String()

	// Filter for swap-related events
	if ds.isSwapRelated(logs) {
		swapEvent := domain.SwapEventData{
			Program:   programName,
			Signature: signature,
			Slot:      result.Context.Slot,
			Timestamp: time.Now(),
			Logs:      logs,
		}

		// Parse specific event data based on the program
		swapEvent.ParsedData = ds.parseSwapData(logs, programName)

		ds.displaySwapEvent(swapEvent)

		// Optionally fetch full transaction details
		go ds.fetchTransactionDetails(signature, programName)
	}
}

// isSwapRelated checks if logs contain swap-related keywords
func (ds *DEXSubscriber) isSwapRelated(logs []string) bool {
	swapKeywords := []string{
		"swap",
		"Swap",
		"trade",
		"Trade",
		"exchange",
		"Exchange",
		"buy",
		"sell",
		"Buy",
		"Sell",
		"InstructionData",
	}

	for _, log := range logs {
		logLower := strings.ToLower(log)
		for _, keyword := range swapKeywords {
			if strings.Contains(logLower, strings.ToLower(keyword)) {
				return true
			}
		}
	}
	return false
}

// parseSwapData extracts swap-specific information from logs
func (ds *DEXSubscriber) parseSwapData(logs []string, programName string) map[string]interface{} {
	data := make(map[string]interface{})

	switch programName {
	case "Raydium-AMMV4":
		data = ds.parseRaydiumLogs(logs)
	case "Raydium-CPMM":
		data = ds.parseRaydiumCPMMLogs(logs)
	case "Raydium-CLMM":
		data = ds.parseRaydiumCLMMLogs(logs)
	case "Orca-Whirlpool":
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

// displaySwapEvent prints formatted swap event information
func (ds *DEXSubscriber) displaySwapEvent(event domain.SwapEventData) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("🔄 SWAP DETECTED on %s\n", event.Program)
	fmt.Printf("📝 Signature: %s\n", event.Signature)
	fmt.Printf("📍 Slot: %d\n", event.Slot)
	fmt.Printf("⏰ Time: %s\n", event.Timestamp.Format("2006-01-02 15:04:05"))

	if len(event.ParsedData) > 0 {
		fmt.Printf("📊 Parsed Data:\n")
		for key, value := range event.ParsedData {
			fmt.Printf("   %s: %v\n", key, value)
		}
	}

	fmt.Printf("📜 Raw Logs:\n")
	for i, log := range event.Logs {
		fmt.Printf("   [%d] %s\n", i, log)
	}
	fmt.Println(strings.Repeat("=", 80))
}

// fetchTransactionDetails gets full transaction information
func (ds *DEXSubscriber) fetchTransactionDetails(signature string, programName string) {
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

	fmt.Printf("\n🔍 Transaction Details for %s (%s):\n", signature[:16]+"...", programName)
	fmt.Printf("   Slot: %d\n", tx.Slot)
	fmt.Printf("   Block Time: %v\n", tx.BlockTime)

	if tx.Meta != nil {
		fmt.Printf("   Fee: %d lamports\n", tx.Meta.Fee)
		if tx.Meta.Err != nil {
			fmt.Printf("   Error: %v\n", tx.Meta.Err)
		}

		// Display token balance changes
		if len(tx.Meta.PreTokenBalances) > 0 || len(tx.Meta.PostTokenBalances) > 0 {
			fmt.Printf("   Token Balance Changes:\n")
			for i, preBalance := range tx.Meta.PreTokenBalances {
				if i < len(tx.Meta.PostTokenBalances) {
					postBalance := tx.Meta.PostTokenBalances[i]
					fmt.Printf("     Account %d: %s -> %s\n",
						preBalance.AccountIndex,
						preBalance.UiTokenAmount.Amount,
						postBalance.UiTokenAmount.Amount)
				}
			}
		}
	}
}
