package contract

import (
	"github.com/spacemeshos/go-spacemesh/common"
	"github.com/spacemeshos/go-spacemesh/crypto"
)

type ContractId = common.Hash

func CalcContractId(code []byte) ContractId {
	return crypto.Keccak256Hash(code)
}
