package contract

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_ContractId_CalcContractId_Must_Not_Be_Empty(t *testing.T) {
	assert.Panics(t, func() {
		CalcContractId([]byte{})
	})
}

func Test_ContractId_CalcContractId_Is_Deterministic(t *testing.T) {
	code := []byte{0xAB, 0xAB, 0xAB}

	id1 := CalcContractId(code)
	id2 := CalcContractId(code)

	assert.Equal(t, id1, id2)
}

func Test_ContractId_Two_Different_Contracts_Should_Have_Different_Ids(t *testing.T) {
	code1 := []byte{0xAB, 0xAB, 0xAB}
	code2 := []byte{0xCD, 0xCD, 0xCD}

	id1 := CalcContractId(code1)
	id2 := CalcContractId(code2)

	assert.NotEqual(t, id1, id2)
}
