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
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	CasperSDK "github.com/casper-ecosystem/casper-golang-sdk/sdk"
	"golang.org/x/crypto/blake2b"

	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
)

const (
	gethHTTPTimeout = 120 * time.Second

	maxTraceConcurrency  = int64(16) // nolint:gomnd
	semaphoreTraceWeight = int64(1)  // nolint:gomnd
)

type Client struct {
	RpcClient *CasperSDK.RpcClient
}

// NewClient creates a Client that from the provided url and params.
func NewClient() (*Client, error) {
	RpcClient := CasperSDK.NewRpcClient("http://104.156.254.95:7777/rpc")
	return &Client{RpcClient}, nil
}

// Status returns status information
// for determining node healthiness.
func (ec *Client) Status(ctx context.Context) (
	*RosettaTypes.BlockIdentifier,
	int64,
	[]*RosettaTypes.Peer,
	error,
) {
	blockres, err := ec.RpcClient.GetLatestBlock()
	if err != nil {
		return nil, -1, nil, err
	}

	casper_peers, err := ec.RpcClient.GetPeers()
	if err != nil {
		return nil, -1, nil, err
	}

	rosetta_peers := make([]*RosettaTypes.Peer, len(casper_peers.Peers))
	for i, peerInfo := range casper_peers.Peers {
		rosetta_peers[i] = &RosettaTypes.Peer{
			PeerID: peerInfo.Address,
			Metadata: map[string]interface{}{
				"Node ID": peerInfo.NodeId,
			},
		}
	}

	return &RosettaTypes.BlockIdentifier{
			Hash:  blockres.Hash,
			Index: int64(blockres.Header.Height),
		},
		blockres.Header.Timestamp.UnixNano() / 1e6,
		rosetta_peers,
		nil
}

// Block returns a populated block at the *RosettaTypes.PartialBlockIdentifier.
// If neither the hash or index is populated in the *RosettaTypes.PartialBlockIdentifier,
// the current block is returned.
func (ec *Client) Block(
	ctx context.Context,
	blockIdentifier *RosettaTypes.PartialBlockIdentifier,
) (*RosettaTypes.Block, error) {
	var block CasperSDK.BlockResponse
	var err error
	var block_transfers []CasperSDK.TransferResponse

	if blockIdentifier != nil {
		if blockIdentifier.Hash != nil {
			block, err = ec.RpcClient.GetBlockByHash(*blockIdentifier.Hash)
			if err != nil {
				return nil, fmt.Errorf("%w: could not get block by hash", err)
			}

			block_transfers, err = ec.RpcClient.GetBlockTransfersByHash(*blockIdentifier.Hash)
			if err != nil {
				return nil, fmt.Errorf("%w: could not get block transfer by hash", err)
			}
		}
		if blockIdentifier.Index != nil {
			var index uint64
			index = uint64(*blockIdentifier.Index)
			block, err = ec.RpcClient.GetBlockByHeight(index)
			if err != nil {
				return nil, fmt.Errorf("%w: could not get block by height", err)
			}

			block_transfers, err = ec.RpcClient.GetBlockTransfersByHeight(uint64(*blockIdentifier.Index))
			if err != nil {
				return nil, fmt.Errorf("%w: could not get block transfer by height", err)
			}
		}
	}
	if blockIdentifier == nil {
		block, err = ec.RpcClient.GetLatestBlock()
		if err != nil {
			return nil, fmt.Errorf("%w: could not get block", err)
		}

		block_transfers, err = ec.RpcClient.GetLatestBlockTransfers()
		if err != nil {
			return nil, fmt.Errorf("%w: could not get block", err)
		}
	}

	BlockIdentifier := &RosettaTypes.BlockIdentifier{
		Hash:  block.Hash,
		Index: int64(block.Header.Height),
	}
	ParentBlockIdentifier := BlockIdentifier
	if BlockIdentifier.Index != GenesisBlockIndex {
		ParentBlockIdentifier = &RosettaTypes.BlockIdentifier{
			Hash:  block.Header.ParentHash,
			Index: BlockIdentifier.Index - 1,
		}
	}

	//reading deploy hash list
	var deployToTransferMap = make(map[string]CasperSDK.TransferResponse)
	for _, trs := range block_transfers {
		if _, ok := deployToTransferMap[trs.DeployHash]; !ok {
			deployToTransferMap[trs.DeployHash] = trs
		}
	}

	Transactions := make(
		[]*RosettaTypes.Transaction,
		len(deployToTransferMap),
	)
	var i = 0
	for _, tx := range deployToTransferMap {
		transaction, _ := ec.CreateRosTransaction(tx)
		Transactions[i] = transaction
		i++
	}

	return &RosettaTypes.Block{
		BlockIdentifier:       BlockIdentifier,
		ParentBlockIdentifier: ParentBlockIdentifier,
		Timestamp:             block.Header.Timestamp.UnixNano() / 1e6,
		Transactions:          Transactions,
	}, nil
}

func (ec *Client) BlockTransaction(
	ctx context.Context,
	blockIdentifier *RosettaTypes.BlockIdentifier,
	transactionIdentifier *RosettaTypes.TransactionIdentifier,
) (*RosettaTypes.Transaction, error) {
	var block_transfers []CasperSDK.TransferResponse
	var err error
	if blockIdentifier != nil {
		if blockIdentifier.Hash != "" {
			block_transfers, err = ec.RpcClient.GetBlockTransfersByHash(blockIdentifier.Hash)
			if err != nil {
				return nil, fmt.Errorf("%w: could not get block", err)
			}
		}
		if blockIdentifier.Index != 0 {
			block_transfers, err = ec.RpcClient.GetBlockTransfersByHeight(uint64(blockIdentifier.Index))
			if err != nil {
				return nil, fmt.Errorf("%w: could not get block", err)
			}
		}
	}
	if blockIdentifier == nil {
		block_transfers, err = ec.RpcClient.GetLatestBlockTransfers()
		if err != nil {
			return nil, fmt.Errorf("%w: could not get block", err)
		}
	}
	var transaction *RosettaTypes.Transaction
	for _, tx := range block_transfers {
		if tx.DeployHash == transactionIdentifier.Hash {
			transaction, _ = ec.CreateRosTransaction(tx)
		}
	}
	return transaction, nil
}

func (ec *Client) CreateRosTransaction(tx CasperSDK.TransferResponse) (*RosettaTypes.Transaction, *RosettaTypes.Error) {
	rosOperations, _ := ec.CreateOperation(tx)
	transaction := &RosettaTypes.Transaction{
		TransactionIdentifier: &RosettaTypes.TransactionIdentifier{
			Hash: tx.DeployHash,
		},
		Operations: rosOperations,
		Metadata:   map[string]interface{}{},
	}

	return transaction, nil

}

func (ec *Client) CreateOperation(tx CasperSDK.TransferResponse) ([]*RosettaTypes.Operation, error) {
	if tx.From == "" {
		tx.From = "account-hash-0000000000000000000000000000000000000000000000000000000000000000"
	}
	if tx.To == "" {
		tx.To = "account-hash-0000000000000000000000000000000000000000000000000000000000000000"
	}
	Neg_Amount := "-" + tx.Amount
	// Neg_Amount = fmt.Sprintf("-%s", tx.Amount)
	var OpStatus string
	deploy, err := ec.RpcClient.GetDeploy(tx.DeployHash)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get deploy", err)
	}
	len := len(deploy.ExecutionResults)
	for i := 0; i < len; i++ {
		var k = []string{}
		k = deploy.ExecutionResults[i].Result.Success.Transfers
		if k == nil {
			OpStatus = FailureStatus
		} else {
			OpStatus = SuccessStatus
		}
	}

	return []*RosettaTypes.Operation{
		{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: 0,
			},
			Type:   TransferOpType,
			Status: RosettaTypes.String(OpStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: tx.From,
			},
			Amount: &RosettaTypes.Amount{
				Value:    Neg_Amount,
				Currency: Currency,
			},
		},

		{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: 1,
			},
			RelatedOperations: []*RosettaTypes.OperationIdentifier{
				{
					Index: 0,
				},
			},
			Type:   TransferOpType,
			Status: RosettaTypes.String(OpStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: tx.To,
			},
			Amount: &RosettaTypes.Amount{
				Value:    tx.Amount,
				Currency: Currency,
			},
		},
	}, nil

}

// Balance returns the balance of a *RosettaTypes.AccountIdentifier
// at a *RosettaTypes.PartialBlockIdentifier.
//
// We must use graphql to get the balance atomically (the
// rpc method for balance does not allow for querying
// by block hash nor return the block hash where
// the balance was fetched).
func (ec *Client) Balance(
	ctx context.Context,
	account *RosettaTypes.AccountIdentifier,
	block *RosettaTypes.PartialBlockIdentifier,
) (*RosettaTypes.AccountBalanceResponse, error) {
	var blockres CasperSDK.BlockResponse
	var err error
	var balance big.Int
	if block != nil {
		if block.Hash != nil {
			blockres, err = ec.RpcClient.GetBlockByHash(*block.Hash)
			if err != nil {
				return nil, fmt.Errorf("%w: could not get block", err)
			}
		}
		if block.Index != nil {
			blockres, err = ec.RpcClient.GetBlockByHeight(uint64(*block.Index))
			if err != nil {
				return nil, fmt.Errorf("%w: could not get block", err)
			}
		}
	}
	if block == nil {
		blockres, err = ec.RpcClient.GetLatestBlock()
		if err != nil {
			return nil, fmt.Errorf("%w: could not get block", err)
		}
	}
	stateRootHash := blockres.Header.StateRootHash

	if strings.Contains(account.Address, "account-hash") {
		var path []string
		item, err := ec.RpcClient.GetStateItem(stateRootHash, account.Address, path)
		if err != nil {
			return nil, fmt.Errorf("%w: could not get state item", err)
		}
		balanceUref := item.Account.MainPurse
		balance, err = ec.RpcClient.GetAccountBalance(stateRootHash, balanceUref)
		if err != nil {
			return nil, fmt.Errorf("can't get account balance")
		}
	} else if strings.Contains(account.Address, "uref") {
		balance, err = ec.RpcClient.GetAccountBalance(stateRootHash, account.Address)
		if err != nil {
			return nil, fmt.Errorf("can't get account balance")
		}
	} else if account.Address[0:2] == "01" || account.Address[0:2] == "02" {
		account := account.Address[2:len(account.Address)]
		pubbyte, _ := hex.DecodeString(account)
		name := strings.ToLower("ED25519")
		sep := "00"
		decoded_sep, _ := hex.DecodeString(sep)
		buffer := append([]byte(name), decoded_sep...)
		buffer = append(buffer, pubbyte...)

		hash := blake2b.Sum256(buffer)
		resHash := fmt.Sprintf("account-hash-%s", hex.EncodeToString(hash[:]))
		var path []string
		item, err := ec.RpcClient.GetStateItem(stateRootHash, resHash, path)
		if err != nil {
			return nil, fmt.Errorf("%w: could not get state item", err)
		}
		balanceUref := item.Account.MainPurse
		balance, err = ec.RpcClient.GetAccountBalance(stateRootHash, balanceUref)
		if err != nil {
			return nil, fmt.Errorf("can't get account balance")
		}
	} else {
		return nil, fmt.Errorf("Invalid input account")
	}

	return &RosettaTypes.AccountBalanceResponse{
		Balances: []*RosettaTypes.Amount{
			{
				Value:    balance.String(),
				Currency: Currency,
			},
		},
		BlockIdentifier: &RosettaTypes.BlockIdentifier{
			Hash:  blockres.Hash,
			Index: int64(blockres.Header.Height),
		},
		Metadata: map[string]interface{}{},
	}, nil
}
