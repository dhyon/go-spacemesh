package contract

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_ContractRegistry_AddContract_AndThen_GetContract(t *testing.T) {
	code := []byte{0xAB, 0xCD, 0xEF}
	id := CalcContractId(code)

	contract := NewContract(id, "Contract #1", code)
	registry := NewContractRegistry()
	registry.AddContract(contract)

	contract_ := registry.GetContractById(id)

	assert.Equal(t, contract, contract_)
}

func Test_ContractRegistry_With_Two_Different_Contracts(t *testing.T) {
	code1 := []byte{0xAB, 0xAB, 0xAB}
	code2 := []byte{0xCD, 0xCD, 0xCD}

	id1 := CalcContractId(code1)
	id2 := CalcContractId(code2)

	contract1 := NewContract(id1, "Contract #1", code1)
	contract2 := NewContract(id2, "Contract #2", code2)
	assert.NotEqual(t, contract1, contract2)

	registry := NewContractRegistry()
	registry.AddContract(contract1)
	registry.AddContract(contract2)

	contract1_ := registry.GetContractById(id1)
	contract2_ := registry.GetContractById(id2)

	assert.Equal(t, contract1, contract1_)
	assert.Equal(t, contract2, contract2_)
	assert.NotEqual(t, contract1_, contract2_)
}
