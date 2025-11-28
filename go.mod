module tapir

go 1.23

replace github.com/dkales/dpf-go => ./modules/dpf-go

require (
	github.com/IBM/mathlib v0.0.2
	github.com/consensys/gnark-crypto v0.12.1
	github.com/dkales/dpf-go v0.0.0-20210304170054-6eae87348848
	github.com/lukechampine/fastxor v0.0.0-20210322201628-b664bed5a5cc
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.10.0
	github.com/ugorji/go/codec v1.2.12
	lukechampine.com/blake3 v1.3.0
)

require (
	github.com/bits-and-blooms/bitset v1.10.0 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/hyperledger/fabric-amcl v0.0.0-20210603140002-2670f91851c8 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)
