package contract

type Contract struct {
	Id ContractId
	Name string
	Code []byte
}

func NewContract(id ContractId, name string, code []byte) *Contract {
	if len(name) == 0 {
		panic("contract name can't be empty")
	}

	contract := Contract { Id: id, Name: name, Code: code }
	return &contract
}
