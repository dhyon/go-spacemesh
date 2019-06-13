package types

type ExecutionReceipt struct {
	Success bool
	GasUsed uint64
}

func NewExecutionReceipt(success bool, gasUsed uint64) *ExecutionReceipt {
	receipt := ExecutionReceipt { Success: success, GasUsed: gasUsed }

	return &receipt
}
