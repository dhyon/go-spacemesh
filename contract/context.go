package contract

import (
	"math/big"
	"github.com/spacemeshos/go-spacemesh/address"
)

type Context struct {
	ContractId *ContractId
	Sender     *address.Address
	Function   string
	Args       []interface{}
	Amount	   *big.Int
	// GasLeft  big.Int
	// GaLimit	big.Int
}
