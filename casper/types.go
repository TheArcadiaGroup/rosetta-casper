// Copyright 2020 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package casper

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	// NodeVersion
	NodeVersion = "1_2_0"

	// Blockchain is Ethereum.
	Blockchain string = "Casper"

	// MainnetNetwork is the value of the network
	// in MainnetNetworkIdentifier.
	MainnetNetwork string = "casper"

	// TestnetNetwork is the value of the network
	// in TestnetNetworkIdentifier.
	TestnetNetwork string = "casper-test"

	// Symbol is the symbol value
	// used in Currency.
	Symbol = "CSPR"

	// Decimals is the decimals value
	// used in Currency.
	Decimals = 9

	// Transfer type
	TransferOpType = "TRANSFER"

	// FeeOpType is used to represent fee operations.
	FeeOpType = "FEE"

	// SuccessStatus is the status of any
	// Ethereum operation considered successful.
	SuccessStatus = "SUCCESS"

	// FailureStatus is the status of any
	// Ethereum operation considered unsuccessful.
	FailureStatus = "FAILURE"

	// HistoricalBalanceSupported is whether
	// historical balance is supported.
	HistoricalBalanceSupported = true

	// UnclesRewardMultiplier is the uncle reward
	// multiplier.
	UnclesRewardMultiplier = 32

	// MaxUncleDepth is the maximum depth for
	// an uncle to be rewarded.
	MaxUncleDepth = 8

	// GenesisBlockIndex is the index of the
	// genesis block.
	GenesisBlockIndex = int64(0)

	// TransferGasLimit is the gas limit
	// of a transfer.
	TransferGasLimit = int64(21000) //nolint:gomnd

	// // MainnetGethArguments are the arguments to start a mainnet geth instance.
	// MainnetGethArguments = `--config=/app/ethereum/geth.toml --gcmode=archive --graphql`

	// IncludeMempoolCoins does not apply to rosetta-ethereum as it is not UTXO-based.
	IncludeMempoolCoins = false

	MainnetGenesisHash = "2fe9630b7790852e4409d815b04ca98f37effcdf9097d317b9b9b8ad658f47c8"

	TestnetGenesisHash = "7952a42ee8568532bc454498a6d5f303423f7cf272880f37af9222a1448efa43"
)

var (
	// MainnetGenesisBlockIdentifier is the *types.BlockIdentifier
	// of the mainnet genesis block.
	MainnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash:  MainnetGenesisHash,
		Index: GenesisBlockIndex,
	}

	// TestnetGenesisBlockIdentifier is the *types.BlockIdentifier
	// of the testnet genesis block.
	TestnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash:  TestnetGenesisHash,
		Index: GenesisBlockIndex,
	}

	// Currency is the *types.Currency for all
	// Ethereum networks.
	Currency = &types.Currency{
		Symbol:   Symbol,
		Decimals: Decimals,
	}

	// OperationTypes are all suppoorted operation types.
	OperationTypes = []string{
		TransferOpType,
		FeeOpType,
	}

	// OperationStatuses are all supported operation statuses.
	OperationStatuses = []*types.OperationStatus{
		{
			Status:     SuccessStatus,
			Successful: true,
		},
		{
			Status:     FailureStatus,
			Successful: false,
		},
	}

	// CallMethods are all supported call methods.
	CallMethods = []string{}
)
