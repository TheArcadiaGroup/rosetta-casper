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

package services

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/TheArcadiaGroup/rosetta-casper/configuration"
	"github.com/casper-ecosystem/casper-golang-sdk/keypair"
	"github.com/casper-ecosystem/casper-golang-sdk/keypair/ed25519"
	"github.com/casper-ecosystem/casper-golang-sdk/keypair/secp256k1"

	"github.com/TheArcadiaGroup/rosetta-casper/casper/casper_client_sdk"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// ConstructionAPIService implements the server.ConstructionAPIServicer interface.
type ConstructionAPIService struct {
	config *configuration.Configuration
	client Client
}

const (
	CHAIN_NAME      = "Chain Name"
	TRANSFER_AMOUNT = "Transfer Amount"
	PAYMENT_AMOUNT  = "Payment Amount"
	TARGET_ADDR     = "Target ADDR"
	SRC_ADDR        = "Source ADDR"
	GAS_PRICE       = "Gas Price"
	TRANSFER_ID     = "Transfer ID"
)

// NewConstructionAPIService creates a new instance of a ConstructionAPIService.
func NewConstructionAPIService(
	cfg *configuration.Configuration,
	client Client,
) *ConstructionAPIService {
	return &ConstructionAPIService{
		config: cfg,
		client: client,
	}
}

// ConstructionDerive implements the /construction/derive endpoint.
func (s *ConstructionAPIService) ConstructionDerive(
	ctx context.Context,
	request *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	resp := &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{},
	}
	if request.PublicKey.CurveType == keypair.StrKeyTagEd25519 {
		resp.AccountIdentifier.Address = ed25519.AccountHex(request.PublicKey.Bytes)
	}
	if request.PublicKey.CurveType == keypair.StrKeyTagSecp256k1 {
		resp.AccountIdentifier.Address = secp256k1.AccountHex(request.PublicKey.Bytes)
	}
	return resp, nil
}

// ConstructionPreprocess implements the /construction/preprocess
// endpoint.
func (s *ConstructionAPIService) ConstructionPreprocess(
	ctx context.Context,
	request *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	preProcessResp := &types.ConstructionPreprocessResponse{
		Options:            make(map[string]interface{}),
		RequiredPublicKeys: []*types.AccountIdentifier{},
	}
	preProcessResp.Options[CHAIN_NAME] = request.NetworkIdentifier.Network
	for _, operation := range request.Operations {
		if operation.OperationIdentifier.Index == 0 {
			preProcessResp.Options[SRC_ADDR] = operation.Account.Address
			// preProcessResp.Options[TRANSFER_AMOUNT] = operation.Amount.Value
			sender := &types.AccountIdentifier{
				Address: operation.Account.Address,
			}
			// to request sender public key, so we can remove pubKey from account identifier's meta data
			preProcessResp.RequiredPublicKeys = append(preProcessResp.RequiredPublicKeys, sender)
		}
		if operation.OperationIdentifier.Index == 1 {
			preProcessResp.Options[TRANSFER_AMOUNT] = operation.Amount.Value
			preProcessResp.Options[TARGET_ADDR] = operation.Account.Address
			preProcessResp.Options[TRANSFER_ID] = operation.Metadata[TRANSFER_ID]
		}
	}
	preProcessResp.Options[GAS_PRICE] = "1"
	preProcessResp.Options[PAYMENT_AMOUNT] = request.MaxFee[0].Value

	return preProcessResp, nil
}

// ConstructionMetadata implements the /construction/metadata endpoint.
func (s *ConstructionAPIService) ConstructionMetadata(
	ctx context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, ErrUnavailableOffline
	}

	resp := &types.ConstructionMetadataResponse{
		Metadata:     make(map[string]interface{}),
		SuggestedFee: []*types.Amount{},
	}

	resp.Metadata[CHAIN_NAME] = request.Options[CHAIN_NAME]
	resp.Metadata[TRANSFER_AMOUNT] = request.Options[TRANSFER_AMOUNT]
	resp.Metadata[TARGET_ADDR] = request.Options[TARGET_ADDR]
	resp.Metadata[SRC_ADDR] = request.Options[SRC_ADDR]
	resp.Metadata[GAS_PRICE] = request.Options[GAS_PRICE]
	resp.Metadata[PAYMENT_AMOUNT] = request.Options[PAYMENT_AMOUNT]
	resp.Metadata[TRANSFER_ID] = request.Options[TRANSFER_ID]

	return resp, nil
}

// ConstructionPayloads implements the /construction/payloads endpoint.
func (s *ConstructionAPIService) ConstructionPayloads(
	ctx context.Context,
	request *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	resp := new(types.ConstructionPayloadsResponse)
	payloads := make([]*types.SigningPayload, 0)
	transfer_id, err := strconv.ParseInt(request.Metadata[TRANSFER_ID].(string), 10, 64)
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}
	deployParams := &casper_client_sdk.DeployParams{
		ChainName:      request.Metadata[CHAIN_NAME].(string),
		TransferAmount: request.Metadata[TRANSFER_AMOUNT].(string),
		PaymentAmount:  request.Metadata[PAYMENT_AMOUNT].(string),
		TargetAccount:  request.Metadata[TARGET_ADDR].(string),
		SrcAccount:     request.Metadata[SRC_ADDR].(string),
		GasPrice:       request.Metadata[GAS_PRICE].(string),
		TransferID:     transfer_id,
	}
	unsignTransferJson, err := json.Marshal(casper_client_sdk.NewDeploy(*deployParams))
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}
	resp.UnsignedTransaction = string(unsignTransferJson)
	targetAccount := &types.AccountIdentifier{
		Address: request.Metadata[TARGET_ADDR].(string),
		// Metadata: map[string]interface{}{
		// 	rosettaUtil.Base16: rosettaUtil.ToChecksumAddr(transactionJson[rosettaUtil.SENDER_ADDR].(string)),
		// },
	}
	signingPayload := &types.SigningPayload{
		AccountIdentifier: targetAccount,
		// Bytes:             unsignedByte, //byte array of transaction
		// SignatureType:     rosettaUtil.SIGNATURE_TYPE,
	}
	payloads = append(payloads, signingPayload)
	resp.Payloads = payloads

	return resp, nil
}

// ConstructionCombine implements the /construction/combine
// endpoint.
func (s *ConstructionAPIService) ConstructionCombine(
	ctx context.Context,
	request *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {
	var unsignedTx casper_client_sdk.Deploy
	if err := json.Unmarshal([]byte(request.UnsignedTransaction), &unsignedTx); err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	return &types.ConstructionCombineResponse{
		// SignedTransactiosn: string(signedTxJSON),
	}, nil
}

// ConstructionHash implements the /construction/hash endpoint.
func (s *ConstructionAPIService) ConstructionHash(
	ctx context.Context,
	request *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	// signedTx := ethTypes.Transaction{}
	// if err := signedTx.UnmarshalJSON([]byte(request.SignedTransaction)); err != nil {
	// 	return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	// }

	// hash := signedTx.Hash().Hex()

	// return &types.TransactionIdentifierResponse{
	// 	TransactionIdentifier: &types.TransactionIdentifier{
	// 		Hash: hash,
	// 	},
	// }, nil
	return nil, wrapErr(ErrUnimplemented, nil)
}

// ConstructionParse implements the /construction/parse endpoint.
func (s *ConstructionAPIService) ConstructionParse(
	ctx context.Context,
	request *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	// var tx transaction
	// if !request.Signed {
	// 	err := json.Unmarshal([]byte(request.Transaction), &tx)
	// 	if err != nil {
	// 		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	// 	}
	// } else {
	// 	t := new(ethTypes.Transaction)
	// 	err := t.UnmarshalJSON([]byte(request.Transaction))
	// 	if err != nil {
	// 		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	// 	}

	// 	tx.To = t.To().String()
	// 	tx.Value = t.Value()
	// 	tx.Input = t.Data()
	// 	tx.Nonce = t.Nonce()
	// 	tx.GasPrice = t.GasPrice()
	// 	tx.GasLimit = t.Gas()
	// 	tx.ChainID = t.ChainId()

	// 	msg, err := t.AsMessage(ethTypes.NewEIP155Signer(t.ChainId()))
	// 	if err != nil {
	// 		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	// 	}

	// 	tx.From = msg.From().Hex()
	// }

	// // Ensure valid from address
	// checkFrom, ok := ethereum.ChecksumAddress(tx.From)
	// if !ok {
	// 	return nil, wrapErr(ErrInvalidAddress, fmt.Errorf("%s is not a valid address", tx.From))
	// }

	// // Ensure valid to address
	// checkTo, ok := ethereum.ChecksumAddress(tx.To)
	// if !ok {
	// 	return nil, wrapErr(ErrInvalidAddress, fmt.Errorf("%s is not a valid address", tx.To))
	// }

	// ops := []*types.Operation{
	// 	{
	// 		Type: ethereum.CallOpType,
	// 		OperationIdentifier: &types.OperationIdentifier{
	// 			Index: 0,
	// 		},
	// 		Account: &types.AccountIdentifier{
	// 			Address: checkFrom,
	// 		},
	// 		Amount: &types.Amount{
	// 			Value:    new(big.Int).Neg(tx.Value).String(),
	// 			Currency: ethereum.Currency,
	// 		},
	// 	},
	// 	{
	// 		Type: ethereum.CallOpType,
	// 		OperationIdentifier: &types.OperationIdentifier{
	// 			Index: 1,
	// 		},
	// 		RelatedOperations: []*types.OperationIdentifier{
	// 			{
	// 				Index: 0,
	// 			},
	// 		},
	// 		Account: &types.AccountIdentifier{
	// 			Address: checkTo,
	// 		},
	// 		Amount: &types.Amount{
	// 			Value:    tx.Value.String(),
	// 			Currency: ethereum.Currency,
	// 		},
	// 	},
	// }

	// metadata := &parseMetadata{
	// 	Nonce:    tx.Nonce,
	// 	GasPrice: tx.GasPrice,
	// 	ChainID:  tx.ChainID,
	// }
	// metaMap, err := marshalJSONMap(metadata)
	// if err != nil {
	// 	return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	// }

	// var resp *types.ConstructionParseResponse
	// if request.Signed {
	// 	resp = &types.ConstructionParseResponse{
	// 		Operations: ops,
	// 		AccountIdentifierSigners: []*types.AccountIdentifier{
	// 			{
	// 				Address: checkFrom,
	// 			},
	// 		},
	// 		Metadata: metaMap,
	// 	}
	// } else {
	// 	resp = &types.ConstructionParseResponse{
	// 		Operations:               ops,
	// 		AccountIdentifierSigners: []*types.AccountIdentifier{},
	// 		Metadata:                 metaMap,
	// 	}
	// }
	// return resp, nil
	return nil, wrapErr(ErrUnimplemented, nil)
}

// ConstructionSubmit implements the /construction/submit endpoint.
func (s *ConstructionAPIService) ConstructionSubmit(
	ctx context.Context,
	request *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	// if s.config.Mode != configuration.Online {
	// 	return nil, ErrUnavailableOffline
	// }

	// var signedTx ethTypes.Transaction
	// if err := signedTx.UnmarshalJSON([]byte(request.SignedTransaction)); err != nil {
	// 	return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	// }

	// if err := s.client.SendTransaction(ctx, &signedTx); err != nil {
	// 	return nil, wrapErr(ErrBroadcastFailed, err)
	// }

	// txIdentifier := &types.TransactionIdentifier{
	// 	Hash: signedTx.Hash().Hex(),
	// }
	// return &types.TransactionIdentifierResponse{
	// 	TransactionIdentifier: txIdentifier,
	// }, nil
	return nil, wrapErr(ErrUnimplemented, nil)
}
