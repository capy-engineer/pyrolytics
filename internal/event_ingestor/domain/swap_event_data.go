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
