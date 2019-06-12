package contract

type InMemoryContractRegistry struct {
	contracts map[ContractId]*Contract
}

func NewInMemoryContractRegistry() ContractRegistry {
	registry := InMemoryContractRegistry { contracts: make(map[ContractId]*Contract) }
	return &registry
}

func (registry *InMemoryContractRegistry) AddContract(contract *Contract) {
	registry.contracts[contract.Id] = contract
}

func (registry *InMemoryContractRegistry) GetContractById(id *ContractId) *Contract {
	return registry.contracts[*id]
}
