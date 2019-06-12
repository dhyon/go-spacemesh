package contract

import (
	"path"
	"io/ioutil"
	"math/big"
	// "github.com/stretchr/testify/assert"
	"testing"

	"github.com/spacemeshos/go-spacemesh/common"
	"github.com/spacemeshos/go-spacemesh/database"
	"github.com/spacemeshos/go-spacemesh/state"
	"github.com/spacemeshos/go-spacemesh/address"
)

func setupVM() *VM {
	var innerdb database.Database = database.NewMemDatabase()
	var db state.Database = state.NewDatabase(innerdb)
	state, err := state.New(common.Hash{}, db)

	if err != nil {
		panic(err)
	}

	registry := NewInMemoryContractRegistry()

	return NewVM(state, registry)
}

func createCtx(function string, contract *Contract, sender *address.Address, amount *big.Int) *Context {
	ctx := Context {
		Function:	function,
		ContractId: &contract.Id,
		Sender:		sender,
		Amount:		amount,
	}

	return &ctx
}

func storeContractCode(vm *VM, name string) *ContractId {
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

	vm.Registry.AddContract(&contract)

	return &contractId
}

func Test_VM_Stateless_Contract(t *testing.T) {
	vm := setupVM()
	contractId := storeContractCode(vm, "sm_transfer.wast")

	contract := vm.Registry.GetContractById(contractId)

	function := "transfer"
	amount := new(big.Int)
	amount.SetString("1234", 10)

	sender := address.HexToAddress("0xAABBCCDD")
	ctx := createCtx(function, contract, &sender, amount)

	vm.Execute(ctx)
}
