module github.com/TheArcadiaGroup/rosetta-casper

require (
	github.com/OneOfOne/xxhash v1.2.5 // indirect
	github.com/casper-ecosystem/casper-golang-sdk v0.0.0-20210512154135-0e4877acec7b
	github.com/coinbase/rosetta-sdk-go v0.6.5
	github.com/ethereum/go-ethereum v1.9.24
	github.com/fatih/color v1.10.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
)

replace github.com/casper-ecosystem/casper-golang-sdk => github.com/phamvancam2104/casper-golang-sdk v0.0.0-20210625210622-4acb09d72768

go 1.15
