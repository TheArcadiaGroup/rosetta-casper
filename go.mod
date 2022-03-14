module github.com/TheArcadiaGroup/rosetta-casper

require (
	github.com/casper-ecosystem/casper-golang-sdk v0.0.0-20210512154135-0e4877acec7b
	github.com/coinbase/rosetta-sdk-go v0.7.4
	github.com/fatih/color v1.13.0
	github.com/spf13/cobra v1.1.3
	github.com/tendermint/tendermint v0.34.11 // indirect
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

replace github.com/casper-ecosystem/casper-golang-sdk => github.com/phamvancam2104/casper-golang-sdk v0.0.0-20210627230311-0f24d2f9fa54

go 1.15
