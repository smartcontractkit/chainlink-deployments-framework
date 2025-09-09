/*
Package addrconv provides utilities for converting blockchain addresses to bytes across different chain families.

This package implements the Strategy pattern to handle address conversion for various blockchain networks
including EVM, Solana, Aptos, Sui, TON, and TRON. It automatically detects the appropriate converter
based on the chain family and handles the conversion seamlessly.

# Basic Usage

The package provides a single function for address conversion:

- ToBytes - converts addresses using a family string

# Converting with Blockchain Interface

When you have a blockchain object that implements the chain.BlockChain interface,
you can extract the family and use it with ToBytes:

	package main

	import (
		"fmt"
		"log"

		"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
		"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
		"github.com/smartcontractkit/chainlink-deployments-framework/chain/utils/addrconv"
		chain_selectors "github.com/smartcontractkit/chain-selectors"
	)

	func main() {
		// Create blockchain objects
		ethChain := evm.Chain{Selector: chain_selectors.ETHEREUM_MAINNET.Selector}
		solChain := solana.Chain{Selector: chain_selectors.SOLANA_MAINNET.Selector}

		// Convert addresses using the blockchain family
		ethAddress := "0x742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8"
		bytes, err := addrconv.ToBytes(ethChain.Family(), ethAddress)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("EVM Address: %s\n", ethAddress)
		fmt.Printf("Bytes: %x\n", bytes)
		fmt.Printf("Length: %d bytes\n\n", len(bytes))

		// Convert Solana address
		solAddress := "11111111111111111111111111111112"
		bytes, err = addrconv.ToBytes(solChain.Family(), solAddress)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Solana Address: %s\n", solAddress)
		fmt.Printf("Bytes: %x\n", bytes)
		fmt.Printf("Length: %d bytes\n", len(bytes))
	}

# Converting with Family Strings

	package main

	import (
		"fmt"
		"log"

		"github.com/smartcontractkit/chainlink-deployments-framework/chain/utils/addrconv"
		chain_selectors "github.com/smartcontractkit/chain-selectors"
	)

	func main() {
		examples := []struct {
			family  string
			addr    string
			desc    string
		}{
			{
				family: chain_selectors.FamilyEVM,
				addr:   "0x742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8",
				desc:   "Ethereum address",
			},
			{
				family: chain_selectors.FamilySolana,
				addr:   "11111111111111111111111111111112",
				desc:   "Solana System Program",
			},
			{
				family: chain_selectors.FamilyAptos,
				addr:   "0x1",
				desc:   "Aptos framework account",
			},
		}

		for _, example := range examples {
			bytes, err := addrconv.ToBytes(example.family, example.addr)
			if err != nil {
				log.Printf("Error converting %s: %v", example.desc, err)
				continue
			}

			fmt.Printf("%s (%s):\n", example.desc, example.family)
			fmt.Printf("  Address: %s\n", example.addr)
			fmt.Printf("  Bytes: %x\n", bytes)
			fmt.Printf("  Length: %d bytes\n\n", len(bytes))
		}
	}

# Supported Chain Families

The package supports the following blockchain families:

	EVM (Ethereum Virtual Machine):
	  - Family: chain_selectors.FamilyEVM
	  - Address format: 0x prefixed hex (20 bytes)
	  - Examples: Ethereum, Polygon, BSC, Avalanche
	  - Sample: "0x742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8"

	Solana:
	  - Family: chain_selectors.FamilySolana
	  - Address format: Base58 encoded (32 bytes)
	  - Sample: "11111111111111111111111111111112"

	Aptos:
	  - Family: chain_selectors.FamilyAptos
	  - Address format: 0x prefixed hex, variable length (32 bytes normalized)
	  - Samples: "0x1", "0x0000...0001"

	Sui:
	  - Family: chain_selectors.FamilySui
	  - Address format: 0x prefixed hex (32 bytes)
	  - Sample: "0xa402ce953053607dffcdfec89406c579c8d8ddb9c90e01b7aa28f5f1538ac289"

	TON:
	  - Family: chain_selectors.FamilyTon
	  - Address format: Base64 encoded (32 bytes)
	  - Sample: "EQAAAQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHx2j"

	TRON:
	  - Family: chain_selectors.FamilyTron
	  - Address format: Base58 encoded (21 bytes)
	  - Sample: "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"
*/
package addrconv
