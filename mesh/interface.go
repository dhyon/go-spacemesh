package mesh

import (
	"context"

	"github.com/spacemeshos/go-spacemesh/common/types"
)

//go:generate mockgen -package=mocks -destination=./mocks/mocks.go -source=./interface.go

type state interface {
	ApplyLayer(layer types.LayerID, txs []*types.Transaction, rewards map[types.Address]uint64) ([]*types.Transaction, error)
	GetStateRoot() types.Hash32
	Rewind(layer types.LayerID) (types.Hash32, error)
	AddTxToPool(tx *types.Transaction) error

	// below APIs exist to satisfy TxAPI interface

	AddressExists(types.Address) bool
	GetAllAccounts() (*types.MultipleAccountsState, error)
	GetBalance(types.Address) uint64
	GetLayerApplied(types.TransactionID) *types.LayerID
	GetLayerStateRoot(types.LayerID) (types.Hash32, error)
	GetNonce(types.Address) uint64
	ValidateNonceAndBalance(*types.Transaction) error
}

type tortoise interface {
	OnBallot(*types.Ballot)
	OnBlock(*types.Block)
	HandleIncomingLayer(context.Context, types.LayerID) (oldPbase, newPbase types.LayerID, reverted bool)
}
