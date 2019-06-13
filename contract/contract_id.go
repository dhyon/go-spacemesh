package contract

import (
	"github.com/spacemeshos/go-spacemesh/address"
	"github.com/spacemeshos/go-spacemesh/crypto"
)

type ContractId = address.Address

func CalcContractId(code []byte) *ContractId {
	if len(code) == 0 {
		panic("code must not be empty")
	}

	bytes := crypto.Keccak256(code)

	id := address.BytesToAddress(bytes)
	return &id
}
