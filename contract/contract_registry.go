package contract

type ContractRegistry struct {
	contracts map[ContractId]*Contract
}

func NewContractRegistry() *ContractRegistry {
	registry := ContractRegistry { contracts: make(map[ContractId]*Contract) }
	return &registry
}

func (registry *ContractRegistry) AddContract(contract *Contract) {
	registry.contracts[contract.Id] = contract
}

func (registry *ContractRegistry) GetContractById(id ContractId) *Contract {
	return registry.contracts[id]
}
