package contract

// #include <stdlib.h>
//
// extern int32_t sm_vm_get_sender_addr(void *context);
// extern int32_t sm_vm_get_balance(void *context, int32_t addrPtr, int32_t addrLen);
// extern void sm_vm_set_balance(void *context, int32_t addrPtr, int32_t addrLen, int32_t balance);
import "C"

import (
	"fmt"
	"unsafe"
	"math/big"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"

	"github.com/spacemeshos/go-spacemesh/common"
	"github.com/spacemeshos/go-spacemesh/state"
	"github.com/spacemeshos/go-spacemesh/types"
	"github.com/spacemeshos/go-spacemesh/address"
	"github.com/spacemeshos/go-spacemesh/contract/gas"
)

type VM struct {
	context  uintptr  // *Context
	state 	 uintptr  // *state.StateDB
	gasUsed  uint64
}

func NewVM(state *state.StateDB) *VM {
	return &VM {gasUsed: 0, state: uintptr(unsafe.Pointer(state))}
}

func (vm *VM) GetState() *state.StateDB {
	unsafePtr := unsafe.Pointer(vm.state)
	return (*state.StateDB)(unsafePtr)
}

func (vm *VM) GetCtx() *Context {
	unsafePtr := unsafe.Pointer(vm.context)
	return (*Context)(unsafePtr)
}

//export sm_vm_get_sender_addr
func sm_vm_get_sender_addr(vmctx unsafe.Pointer) int32 {
	return 0
}

//export sm_vm_get_balance
func sm_vm_get_balance(vmctx unsafe.Pointer, addrPtr int32, addrLen int32) int32 {
	// log := fmt.Sprintf("sm_vm_get_balance with addrPtr=%d, addrLen=%d", addrPtr, addrLen)
	// fmt.Println(log)

	vm, mem := unwrapDataAndMemory(vmctx)
	state := vm.GetState()

	addrBytes := mem.Data()[addrPtr : addrPtr + addrLen]

	addr := address.BytesToAddress(addrBytes)
	amount := state.GetBalance(addr)

	fmt.Println(amount)

	return 0
}

//export sm_vm_set_balance
func sm_vm_set_balance(vmctx unsafe.Pointer, addrPtr int32, addrLen int32, balance int32) {
	log := fmt.Sprintf("sm_vm_set_balance with addrPtr=%d , addrLen=%d , balance=%d ", addrPtr, addrLen, balance)
	fmt.Println(log)
}

func unwrapData(vmctx unsafe.Pointer) *VM {
	var instanceContext = wasm.IntoInstanceContext(vmctx)
	return (*VM)(instanceContext.Data())
}

func unwrapMemory(vmctx unsafe.Pointer) *wasm.Memory {
	var instanceContext = wasm.IntoInstanceContext(vmctx)
	return instanceContext.Memory()
}

func unwrapDataAndMemory(vmctx unsafe.Pointer) (*VM, *wasm.Memory) {
	var instanceContext = wasm.IntoInstanceContext(vmctx)

	vm := (*VM)(instanceContext.Data())
	mem := instanceContext.Memory()

	return vm, mem
}

func allocateFuncArgs(ctx *Context, instance *wasm.Instance) ([]interface{}, error) {
	allocateFunc := instance.Exports["Allocate"]

	if allocateFunc == nil {
		return []interface{}{}, fmt.Errorf("Couldn't find exported function `Allocate`")
	}

	// TODO: calculate `nargs`
	nargs := 4
	args := make([]interface{}, nargs, nargs)

	// allocate the `sender` address first
	res, err := allocateFunc(common.AddressLength)
	if err != nil {
		return nil, err
	}

	args[0] = (interface{})(res.ToI32())

	// allocating the args given in `ctx.Args`
	i := 1
	for _, arg := range ctx.Args {
		switch arg.(type) {
		case address.Address:
			res, err := allocateFunc(common.AddressLength)

			if err != nil {
				return nil, err
			}

			args[i] = (interface{})(res.ToI32())
			i += 1
		case *big.Int:
			bigNum := arg.(*big.Int)
			bigIntAsBytes := bigNum.Bytes()
			bigIntSize := len(bigIntAsBytes)

			res, err := allocateFunc(bigIntSize)

			if err != nil {
				return nil, err
			}

			args[i] = (interface{})(res.ToI32())
			args[i + 1] = bigIntSize
			i += 2
		default:
			fmt.Println("unknown")
			i += 1
		}
	}

	return args, nil
}

func ExecuteContract(vm *VM, registry ContractRegistry, ctx *Context) (*types.ExecutionReceipt, error) {
	contract, err := registry.GetContractById(ctx.ContractId)

	if err != nil {
		receipt := types.NewExecutionReceipt(false, gas.ContractNotFound)
		return receipt, err
	}

	imports := wasm.NewImports().Namespace("env")
	imports, _ = imports.Append("sm_vm_get_sender_addr", sm_vm_get_sender_addr, C.sm_vm_get_sender_addr)
	imports, _ = imports.Append("sm_vm_get_balance", sm_vm_get_balance, C.sm_vm_get_balance)
	imports, _ = imports.Append("sm_vm_set_balance", sm_vm_set_balance, C.sm_vm_set_balance)
	instance, err := wasm.NewInstanceWithImports(contract.Code, imports)
	defer instance.Close()

	instance.SetContextData(unsafe.Pointer(vm))

	if err != nil {
		receipt := types.NewExecutionReceipt(false, gas.InstanceInitFailed)
		return receipt, err
	}

	exportedFunc := instance.Exports[ctx.Function]

	if exportedFunc == nil {
		err := fmt.Errorf("Couldn't find exported function `%s`", ctx.Function)
		receipt := types.NewExecutionReceipt(false, gas.ExportFunctionNotFound)
		return receipt, err
	}

	args, err := allocateFuncArgs(ctx, &instance)

	if err != nil {
		receipt := types.NewExecutionReceipt(false, gas.ExportFunctionArgsAllocationFailed)
		return receipt, err
	}

	_, err = exportedFunc(args...)

	if err == nil {
		receipt := types.NewExecutionReceipt(true, vm.gasUsed)
		return receipt, nil
	} else {
		receipt := types.NewExecutionReceipt(false, vm.gasUsed)
		return receipt, err
	}
}