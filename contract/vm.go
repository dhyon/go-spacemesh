package contract

// #include <stdlib.h>
//
// extern int32_t sm_vm_get_balance(void *context, int32_t addr_ptr, int32_t addr_len);
// extern void sm_vm_set_balance(void *context, int32_t addr_ptr, int32_t addr_len, int32_t balance);
import "C"

import (
	"unsafe"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"

	"github.com/spacemeshos/go-spacemesh/state"
)

type VM struct {
	Context *Context
	State *state.StateDB
	Registry ContractRegistry
}

func NewVM(state *state.StateDB, registry ContractRegistry) *VM {
	return &VM {State: state, Registry: registry}
}

//export sm_vm_get_balance
func sm_vm_get_balance(context unsafe.Pointer, addr_ptr int32, addr_len int32) int32 {
	// account := gs.GetAccount(address)
	// balance := account.GetBalance()
	// fmt.Println(",  balance: ", balance)
	// return (int32)(balance)
	return 0
}

//export sm_vm_set_balance
func sm_vm_set_balance(context unsafe.Pointer, addr_ptr int32, addr_len int32, balance int32) {
	// address := address.Address{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	// account := gs.GetAccount(address)
	// newBalance := (uint32)(balance)
	// account.SetBalance(newBalance)
}

func get_vm(context unsafe.Pointer) *VM {
	var instanceContext = wasm.IntoInstanceContext(context)
	return (*VM)(instanceContext.Data())
}

func (vm *VM) Execute(ctx *Context)  {
	vm.Context = ctx
	contract := vm.Registry.GetContractById(ctx.ContractId)

	imports := wasm.NewImports().Namespace("env")
	imports, _ = imports.Append("get_balance", sm_vm_get_balance, C.sm_vm_get_balance)
	imports, _ = imports.Append("set_balance", sm_vm_set_balance, C.sm_vm_set_balance)

	instance, _ := wasm.NewInstanceWithImports(contract.Code, imports)
	// instance.SetContextData(unsafe.Pointer(&vm))

	// exported_func := instance.Exports[ctx.Function]
	// exported_func()

	defer instance.Close()
}
