module github.com/TheArcadiaGroup/rosetta-casper

require (
	github.com/casper-ecosystem/casper-golang-sdk v0.0.0-20210512154135-0e4877acec7b
	github.com/coinbase/rosetta-sdk-go v0.6.5
	github.com/fatih/color v1.10.0
	github.com/spf13/cobra v1.1.1
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210420205809-ac73e9fd8988 // indirect
)

replace github.com/casper-ecosystem/casper-golang-sdk => github.com/phamvancam2104/casper-golang-sdk v0.0.0-20210625210622-4acb09d72768

go 1.15
