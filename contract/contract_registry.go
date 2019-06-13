package contract

type ContractRegistry interface {
	AddContract(contract *Contract)

	GetContractById(id *ContractId) (*Contract, error)
}
