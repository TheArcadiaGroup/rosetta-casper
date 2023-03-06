module github.com/TheArcadiaGroup/rosetta-casper

require (
	github.com/btcsuite/btcd v0.22.0-beta // indirect
	github.com/casper-ecosystem/casper-golang-sdk v0.0.0-20210512154135-0e4877acec7b
	github.com/coinbase/rosetta-sdk-go v0.6.10
	github.com/fatih/color v1.12.0
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/tendermint/tendermint v0.34.11 // indirect
	golang.org/x/crypto v0.7.0
	golang.org/x/sync v0.1.0
)

replace github.com/casper-ecosystem/casper-golang-sdk => github.com/phamvancam2104/casper-golang-sdk v0.0.0-20210627230311-0f24d2f9fa54

go 1.15
