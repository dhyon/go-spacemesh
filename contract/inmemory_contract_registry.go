package contract

import (
	"fmt"
)

type InMemoryContractRegistry struct {
	contracts map[*ContractId]*Contract
}

func NewInMemoryContractRegistry() ContractRegistry {
	registry := InMemoryContractRegistry { contracts: make(map[*ContractId]*Contract) }
	return &registry
}

func (registry *InMemoryContractRegistry) AddContract(contract *Contract) {
	registry.contracts[contract.Id] = contract
}

func (registry *InMemoryContractRegistry) GetContractById(id *ContractId) (*Contract, error) {
	contract := registry.contracts[id]

	if contract == nil {
		return nil, fmt.Errorf("Contract %s doesn't exit", id)
	} else {
		return contract, nil
	}
}
