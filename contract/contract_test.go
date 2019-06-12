package contract

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Contract_Name_Must_Not_Be_Empty(t *testing.T) {
	code := []byte{0xC0, 0xDE}
	id := CalcContractId(code)

	assert.Panics(t, func() {
		NewContract(id, "", code)
	})
}
