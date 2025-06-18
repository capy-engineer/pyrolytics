package domain

import (
	"time"

	"github.com/gagliardetto/solana-go"
)

var (
	// Raydium Program IDs on Devnet
	RaydiumAMMV4Devnet   = solana.MustPublicKeyFromBase58("HWy1jotHpo6UqeQxx49dpYYdQB8wj9Qk9MdxwjLvDHB8")
	RaydiumCPMMDevnet    = solana.MustPublicKeyFromBase58("CPMDWBwJDtYax9qW7AyRuVC19Cc4L4Vcy4n2BHAbHkCW")
	RaydiumCLMMDevnet    = solana.MustPublicKeyFromBase58("devi51mZmdwUJGU9hjN27vEz64Gps7uUefqxg27EAtH")
	RaydiumRoutingDevnet = solana.MustPublicKeyFromBase58("BVChZ3XFEwTMUk1o9i3HAf91H6mFxSwa5X2wFAWhYPhU")

	// Orca Program IDs (these are for mainnet, devnet addresses may differ)
	OrcaWhirlpoolProgram = solana.MustPublicKeyFromBase58("whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc")
	OrcaWhirlpoolsConfig = solana.MustPublicKeyFromBase58("FcrweFY1G9HJAHG5inkGB6pKg1HZ6x9UC2WioAfWrGkR")
)

type SwapEventData struct {
	Program    string                 `json:"program"`
	Signature  string                 `json:"signature"`
	Slot       uint64                 `json:"slot"`
	Timestamp  time.Time              `json:"timestamp"`
	Logs       []string               `json:"logs"`
	ParsedData map[string]interface{} `json:"parsed_data,omitempty"`
}

type SolanaRawEvent struct {
	// Event metadata
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"` // "swap", "liquidity_add", "liquidity_remove", etc.
	Blockchain string    `json:"blockchain"` // "solana"
	Network    string    `json:"network"`    // "devnet", "mainnet", "testnet"
	Timestamp  time.Time `json:"timestamp"`

	// Transaction information
	Signature string `json:"signature"`
	Slot      uint64 `json:"slot"`
	BlockTime *int64 `json:"block_time,omitempty"`
	Fee       uint64 `json:"fee,omitempty"`
	Success   bool   `json:"success"`

	// DEX information
	DEXName    string `json:"dex_name"`    // "Raydium", "Orca"
	DEXProgram string `json:"dex_program"` // "AMM_V4", "CPMM", "CLMM", "Whirlpool"
	ProgramID  string `json:"program_id"`

	// Swap details (if applicable)
	SwapDetails *SwapDetails `json:"swap_details,omitempty"`

	// Raw data
	RawLogs         []string               `json:"raw_logs"`
	InstructionData map[string]interface{} `json:"instruction_data,omitempty"`

	// Token balance changes
	TokenBalanceChanges []TokenBalanceChange `json:"token_balance_changes,omitempty"`

	// Additional parsed data
	ParsedData map[string]interface{} `json:"parsed_data,omitempty"`
}

type SwapDetails struct {
	Direction      string `json:"direction"` // "buy", "sell"
	AmountIn       string `json:"amount_in"`
	AmountOut      string `json:"amount_out"`
	TokenInMint    string `json:"token_in_mint,omitempty"`
	TokenOutMint   string `json:"token_out_mint,omitempty"`
	TokenInSymbol  string `json:"token_in_symbol,omitempty"`
	TokenOutSymbol string `json:"token_out_symbol,omitempty"`
	Price          string `json:"price,omitempty"`
	Slippage       string `json:"slippage,omitempty"`
	User           string `json:"user,omitempty"`
	Pool           string `json:"pool,omitempty"`
}

type TokenBalanceChange struct {
	AccountIndex uint8  `json:"account_index"`
	Mint         string `json:"mint"`
	Owner        string `json:"owner,omitempty"`
	PreBalance   string `json:"pre_balance"`
	PostBalance  string `json:"post_balance"`
	Change       string `json:"change"`
	Decimals     uint8  `json:"decimals,omitempty"`
}
