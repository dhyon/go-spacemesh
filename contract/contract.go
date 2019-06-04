package contract

type Contract struct {
	Id ContractId
	Name string
	Code []byte
}

func NewContract(id ContractId, name string, code []byte) *Contract {
	contract := Contract { Id: id, Name: name, Code: code }
	return &contract
}
