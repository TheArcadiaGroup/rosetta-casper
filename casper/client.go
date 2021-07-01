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
	"log"
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
	ED25519              = "ed25519"
	SECP256K1            = "secp256k1"
)

type Client struct {
	RpcClient *CasperSDK.RpcClient
}

// NewClient creates a Client that from the provided url and params.
func NewClient() (*Client, error) {
	RpcClient := CasperSDK.NewRpcClient("http://45.32.28.180:7777/rpc")
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

func (ec *Client) GetBlockResponse(
	blockIdentifier *RosettaTypes.PartialBlockIdentifier,
) (*CasperSDK.BlockResponse, error) {
	if blockIdentifier.Hash != nil {
		block, err := ec.RpcClient.GetBlockByHash(*blockIdentifier.Hash)
		if err != nil {
			return nil, fmt.Errorf("%w: could not get block by hash", err)
		}

		return &block, err
	}
	if blockIdentifier.Index != nil {
		var index uint64
		index = uint64(*blockIdentifier.Index)
		block, err := ec.RpcClient.GetBlockByHeight(index)
		if err != nil {
			return nil, fmt.Errorf("%w: could not get block by height", err)
		}

		return &block, err
	}
	return nil, fmt.Errorf("invalid block identifier")
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
	var deployToTransferMap = make(map[string][]*CasperSDK.TransferResponse)
	for _, trs := range block_transfers {
		if _, ok := deployToTransferMap[trs.DeployHash]; !ok {
			deployToTransferMap[trs.DeployHash] = []*CasperSDK.TransferResponse{}
		}
		deployToTransferMap[trs.DeployHash] = append(deployToTransferMap[trs.DeployHash], &trs)
	}

	alldeployHashes := block.Body.DeployHashes
	for _, deployHash := range alldeployHashes {
		if _, ok := deployToTransferMap[deployHash]; !ok {
			deployToTransferMap[deployHash] = []*CasperSDK.TransferResponse{}
		}
	}
	Transactions := make(
		[]*RosettaTypes.Transaction,
		len(deployToTransferMap),
	)

	validator := block.Body.Proposer
	validatorMainPurse, err := ec.GetMainPurseFromPublicKey(validator, block.Header.StateRootHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator purse")
	}

	var i = 0
	for deployHash, transfers := range deployToTransferMap {
		log.Printf("%s %d %s", deployHash, len(transfers), block.Hash)
		transaction, err := ec.CreateRosTransaction(deployHash, transfers, &block, validatorMainPurse)
		if err != nil {
			return nil, fmt.Errorf("%w: Failed to create rosetta transaction for deploy "+deployHash, err)
		}
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
	if transactionIdentifier == nil {
		return nil, fmt.Errorf("null pointer input")
	}
	// var block_transfers []CasperSDK.TransferResponse
	// var err error
	// if blockIdentifier != nil {
	// 	if blockIdentifier.Hash != "" {
	// 		block_transfers, err = ec.RpcClient.GetBlockTransfersByHash(blockIdentifier.Hash)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("%w: could not get block", err)
	// 		}
	// 	}
	// 	if blockIdentifier.Index != 0 {
	// 		block_transfers, err = ec.RpcClient.GetBlockTransfersByHeight(uint64(blockIdentifier.Index))
	// 		if err != nil {
	// 			return nil, fmt.Errorf("%w: could not get block", err)
	// 		}
	// 	}
	// }
	// if blockIdentifier == nil {
	// 	block_transfers, err = ec.RpcClient.GetLatestBlockTransfers()
	// 	if err != nil {
	// 		return nil, fmt.Errorf("%w: could not get block", err)
	// 	}
	// }
	deploy, err := ec.RpcClient.GetDeploy(transactionIdentifier.Hash)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get deploy", err)
	}
	if len(deploy.ExecutionResults) == 0 {
		return nil, fmt.Errorf("invalid deploy")
	}

	blkIdentifier := &RosettaTypes.PartialBlockIdentifier{Hash: &deploy.ExecutionResults[0].BlockHash}
	block, err := ec.GetBlockResponse(blkIdentifier)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get deploy block", err)
	}

	block_transfers, err := ec.RpcClient.GetBlockTransfersByHash(blockIdentifier.Hash)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get block", err)
	}

	var transfers []*CasperSDK.TransferResponse
	for i := 0; i < len(block_transfers); i++ {
		if block_transfers[i].DeployHash == deploy.Deploy.Hash {
			transfers = append(transfers, &block_transfers[i])
		}
	}

	validator := block.Body.Proposer
	validatorMainPurse, err := ec.GetMainPurseFromPublicKey(validator, block.Header.StateRootHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator purse BlockTransaction")
	}

	return ec.CreateRosTransaction(deploy.Deploy.Hash, transfers, block, validatorMainPurse)
}

func PurseWithoutIndex(purse string) string {
	lastIndex := strings.LastIndex(purse, "-")
	return purse[:lastIndex]
}

func (ec *Client) CreateRosTransaction(deployHash string, transfers []*CasperSDK.TransferResponse, block *CasperSDK.BlockResponse, validatorMainPurse string) (*RosettaTypes.Transaction, error) {
	//read deploy
	deploy, err := ec.RpcClient.GetDeploy(deployHash)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get deploy", err)
	}

	if len(deploy.ExecutionResults) == 0 {
		return nil, fmt.Errorf("invalid deploy")
	}

	var rosOperations []*RosettaTypes.Operation
	signerMainPurse, err := ec.GetMainPurseFromPublicKey(deploy.Deploy.Header.Account, block.Header.StateRootHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get signer purse " + deploy.Deploy.Header.Account)
	}
	var paymentAmount string = "0"
	if len(deploy.Deploy.Payment.ModuleBytes.Args) != 0 {
		paymentAmount, err = ec.RpcClient.ReadPaymentAmount(&deploy.Deploy)
		if err != nil {
			return nil, fmt.Errorf("failed to get read payment amount")
		}
	}
	log.Printf("%s %s %s", validatorMainPurse, signerMainPurse, paymentAmount)

	if deploy.ExecutionResults[0].Result.Failure != nil {
		txCost := deploy.ExecutionResults[0].Result.Failure.Cost
		Neg_Amount := "-" + txCost
		//transaction error, tx cost is moved from sender to validator
		rosOperations = []*RosettaTypes.Operation{
			{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: 0,
				},
				Type:   FeeOpType,
				Status: RosettaTypes.String(FailureStatus),
				Account: &RosettaTypes.AccountIdentifier{
					Address: PurseWithoutIndex(signerMainPurse),
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
				Type:   FeeOpType,
				Status: RosettaTypes.String(FailureStatus),
				Account: &RosettaTypes.AccountIdentifier{
					Address: PurseWithoutIndex(validatorMainPurse),
				},
				Amount: &RosettaTypes.Amount{
					Value:    txCost,
					Currency: Currency,
				},
			},
		}
	} else {
		txCost := deploy.ExecutionResults[0].Result.Success.Cost
		if paymentAmount == "0" {
			paymentAmount = txCost
		}
		Neg_Amount := "-" + paymentAmount
		//transaction error, tx cost is moved from sender to validator
		rosOperations = []*RosettaTypes.Operation{
			{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: 0,
				},
				Type:   FeeOpType,
				Status: RosettaTypes.String(SuccessStatus),
				Account: &RosettaTypes.AccountIdentifier{
					Address: PurseWithoutIndex(signerMainPurse),
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
				Type:   FeeOpType,
				Status: RosettaTypes.String(SuccessStatus),
				Account: &RosettaTypes.AccountIdentifier{
					Address: PurseWithoutIndex(validatorMainPurse),
				},
				Amount: &RosettaTypes.Amount{
					Value:    paymentAmount,
					Currency: Currency,
				},
			},
		}
		for i := 0; i < len(transfers); i++ {
			tx := transfers[i]
			transferAmount := tx.Amount
			Neg_Amount := "-" + transferAmount
			rosOp := &RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: int64(2*i + 2),
				},
				Type:   TransferOpType,
				Status: RosettaTypes.String(SuccessStatus),
				Account: &RosettaTypes.AccountIdentifier{
					Address: PurseWithoutIndex(tx.Source),
				},
				Amount: &RosettaTypes.Amount{
					Value:    Neg_Amount,
					Currency: Currency,
				},
			}

			rosOperations = append(rosOperations, rosOp)

			rosOp2 := &RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: int64(2*i + 1 + 2),
				},
				Type:   TransferOpType,
				Status: RosettaTypes.String(SuccessStatus),
				Account: &RosettaTypes.AccountIdentifier{
					Address: PurseWithoutIndex(tx.Target),
				},
				Amount: &RosettaTypes.Amount{
					Value:    transferAmount,
					Currency: Currency,
				},
			}

			rosOperations = append(rosOperations, rosOp2)
		}
	}

	transaction := &RosettaTypes.Transaction{
		TransactionIdentifier: &RosettaTypes.TransactionIdentifier{
			Hash: deployHash,
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
	var balanceUref string
	if strings.Contains(account.Address, "account-hash") {
		var path []string
		item, err := ec.RpcClient.GetStateItem(stateRootHash, account.Address, path)
		if err != nil {
			return nil, fmt.Errorf("%w: could not get state item", err)
		}
		balanceUref = item.Account.MainPurse
	} else if strings.Contains(account.Address, "uref") {
		if 1 == strings.Count(account.Address, "-") {
			balanceUref = account.Address + "-001" //first uref
		} else {
			balanceUref = account.Address
		}
	} else if account.Address[0:2] == "01" {
		balanceUref, err = ec.GetMainPurseFromPublicKey(account.Address, stateRootHash)
		if err != nil {
			return nil, fmt.Errorf("%w: can't get account purse 01", err)
		}
	} else if account.Address[0:2] == "02" {
		balanceUref, err = ec.GetMainPurseFromPublicKey(account.Address, stateRootHash)
		if err != nil {
			return nil, fmt.Errorf("%w: can't get account purse 02", err)
		}
	} else {
		return nil, fmt.Errorf("Invalid input account")
	}

	balance, err = ec.RpcClient.GetAccountBalance(stateRootHash, balanceUref)
	if err != nil {
		return nil, fmt.Errorf("%w: can't get account balance", err)
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
func (ec *Client) GetMainPurseFromPublicKey(accountAddr string, stateroothash string) (string, error) {
	hashtype := SECP256K1
	if accountAddr[0:2] == "01" {
		hashtype = ED25519
	}
	account := accountAddr[2:]
	pubbyte, _ := hex.DecodeString(account)
	name := hashtype
	sep := "00"
	decoded_sep, _ := hex.DecodeString(sep)
	buffer := append([]byte(name), decoded_sep...)
	buffer = append(buffer, pubbyte...)

	hash := blake2b.Sum256(buffer)
	resHash := fmt.Sprintf("account-hash-%s", hex.EncodeToString(hash[:]))
	var path []string
	item, err := ec.RpcClient.GetStateItem(stateroothash, resHash, path)
	if err != nil {
		return "", fmt.Errorf("could not get state item : ----" + err.Error() + "---")
	}
	balanceUref := item.Account.MainPurse
	return balanceUref, nil
}
