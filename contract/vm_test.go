package contract

import (
	"path"
	"io/ioutil"
	"math/big"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/spacemeshos/go-spacemesh/common"
	"github.com/spacemeshos/go-spacemesh/database"
	"github.com/spacemeshos/go-spacemesh/state"
	"github.com/spacemeshos/go-spacemesh/address"
)

func setupContractsRegistry() ContractRegistry {
	return NewInMemoryContractRegistry()
}

func setupVM() *VM {
	var innerdb database.Database = database.NewMemDatabase()
	var db state.Database = state.NewDatabase(innerdb)
	state, err := state.New(common.Hash{}, db)

	if err != nil {
		panic(err)
	}

	return NewVM(state)
}

func createCtx(function string, contract *Contract, sender *address.Address, amount *big.Int) *Context {
	ctx := Context {
		Function:	function,
		ContractId: contract.Id,
		Sender:		sender,
		Amount:		amount,
	}

	return &ctx
}

func storeContractCode(registry ContractRegistry, name string) *ContractId {
	path := path.Join("./precompiled/wasm", name)
	code, err := ioutil.ReadFile(path)

	if err != nil {
		panic("Couldn't load contract " + name)
	}

	contractId := CalcContractId(code)

	contract := Contract {
		Id: contractId,
		Name: name,
		Code: code,
	}

	registry.AddContract(&contract)

	return contractId
}

func loadTestContract(t *testing.T, registry ContractRegistry, name string) *Contract {
	contractId := storeContractCode(registry, name)

	contract, err := registry.GetContractById(contractId)

	assert.Nil(t, err)

	return contract
}

// func Test_VM_Nop_Contract(t *testing.T) {
// 	vm := setupVM()
// 	registry := setupContractsRegistry()
// 	contract := loadTestContract(t, registry, "nop_contract.wasm")
//
// 	sender := address.HexToAddress("0xabcd")
// 	amount := new(big.Int)
//
// 	ctx := createCtx("Execute", contract, &sender, amount)
//
// 	ExecuteContract(vm, registry, ctx)
// }

func Test_VM_Transfer_Contract(t *testing.T) {
 	vm := setupVM()
  	registry := setupContractsRegistry()
  	contract := loadTestContract(t, registry, "transfer_contract.wasm")

 	sender := address.HexToAddress("0xAAAA")
 	amount := big.NewInt(10203040)

 	ctx := createCtx("Transfer", contract, &sender, amount)

 	recipient := address.HexToAddress("0xBBBB")
	ctx.Args = []interface{}{recipient, amount}

	ExecuteContract(vm, registry, ctx)
}
