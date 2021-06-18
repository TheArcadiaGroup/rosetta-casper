package casper_client_sdk

import "time"

// "context"
// "fmt"

// NewDeployHeader creates a new instance of a DeployHeader.
func NewDeploy(
	deployParams DeployParams,
) *Deploy {
	// payment := NewPayment(paymentAmount)
	// session := NewSession(transferAmount, targetAccount, transferID)

	return &Deploy{
		// ModuleBytes: moduleBytes,
	}
}

type DeployParams struct {
	ChainName      string
	TransferAmount string
	PaymentAmount  string
	TargetAccount  string
	SrcAccount     string
	GasPrice       string
	TransferID     int64
}

// NewDeployHeader creates a new instance of a DeployHeader.
func NewDeployHeader(
	account string,
	chainName string,
	bodyHash string,
) *DeployHeader {
	return &DeployHeader{
		Account:      account,
		Timestamp:    time.Now(),
		TTL:          "30m",
		GasPrice:     1,
		BodyHash:     bodyHash,
		Dependencies: nil,
		ChainName:    chainName,
	}
}

// NewPayment creates a new instance of a Payment.
func NewPayment(
	paymentAmount string,
) *Payment {
	var moduleBytes ModuleBytes
	moduleBytes.Module_Bytes = ""
	moduleBytes.Args.Amount.CLType = "U512"
	// moduleBytes.Args.Amount.AmountBytes = paymentAmount
	moduleBytes.Args.Amount.Amount = paymentAmount

	return &Payment{
		ModuleBytes: moduleBytes,
	}
}

// NewSession creates a new instance of a Session.
func NewSession(
	transferAmount string,
	targetAccount string,
	transferID int64,
) *Session {
	var transfer Transfer
	transfer.Args.Amount.CLType = "U512"
	// transfer.Args.Amount.AmountBytes = transferAmount
	transfer.Args.Amount.Amount = transferAmount
	transfer.Args.Target.CLType.ByteArray = 32
	// transfer.Args.Target.AccountBytes =
	transfer.Args.Target.Account = targetAccount
	// transfer.Args.ID.ID_Bytes =
	transfer.Args.ID.ID = transferID

	return &Session{
		Transfer: transfer,
	}
}

// NewDeployHeader creates a new instance of a DeployHeader.
func NewApproval(
	src_account string,
) *Approval {
	return &Approval{
		Signer: src_account,
	}
}

type Deploy struct {
	Hash      string       `json:"hash"`
	Header    DeployHeader `json:"header"`
	Payment   Payment      `json:"payment"`
	Session   Session      `json:"session"`
	Approvals []Approval   `json:"approvals"`
}

type DeployHeader struct {
	Account      string    `json:"account"`
	Timestamp    time.Time `json:"timestamp"`
	TTL          string    `json:"ttl"`
	GasPrice     int       `json:"gas_price"`
	BodyHash     string    `json:"body_hash"`
	Dependencies []string  `json:"dependencies"`
	ChainName    string    `json:"chain_name"`
}

type Payment struct {
	ModuleBytes ModuleBytes `json:"ModuleBytes"`
}

type ModuleBytes struct {
	Module_Bytes string      `json:"module_bytes"`
	Args         PaymentArgs `json:"args"`
}

type PaymentArgs struct {
	Amount StandardAmount `json:"amount"`
}

type StandardAmount struct {
	CLType      string `json:"cl_type"`
	AmountBytes string `json:"bytes"`
	Amount      string `json:"parsed"`
}

type Session struct {
	Transfer Transfer `json:"Transfer"`
}

type Transfer struct {
	Args TransferArgs `json:"args"`
}

type TransferArgs struct {
	Amount StandardAmount `json:"amount"`
	Target TargetAccount  `json:"target"`
	ID     TransferID     `json:"id"`
}

type TargetAccount struct {
	CLType       AccountCLType `json:"cl_type"`
	AccountBytes string        `json:"bytes"`
	Account      string        `json:"parsed"`
}

type TransferID struct {
	CLType   ID_CLType `json:"cl_type"`
	ID_Bytes string    `json:"bytes"`
	ID       int64     `json:"parsed"`
}

type AccountCLType struct {
	ByteArray uint `json:"ByteArray"`
}

type ID_CLType struct {
	Option string `json:"Option"`
}

type Approval struct {
	Signer    string `json:"signer"`
	Signature string `json:"signature"`
}
