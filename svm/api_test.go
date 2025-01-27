package svm

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/database"
	"github.com/spacemeshos/go-spacemesh/log/logtest"
	"github.com/spacemeshos/go-spacemesh/mempool"
	"github.com/spacemeshos/go-spacemesh/p2p"
	"github.com/spacemeshos/go-spacemesh/p2p/pubsub"
	"github.com/spacemeshos/go-spacemesh/signing"
	"github.com/spacemeshos/go-spacemesh/svm/transaction"
)

type ProjectorMock struct {
	nonceDiff   uint64
	balanceDiff uint64
}

func (p *ProjectorMock) GetProjection(addr types.Address, prevNonce, prevBalance uint64) (nonce, balance uint64, err error) {
	return prevNonce + p.nonceDiff, prevBalance - p.balanceDiff, nil
}

type appliedTxsMock struct{}

func (appliedTxsMock) Put(key []byte, value []byte) error { return nil }
func (appliedTxsMock) Delete(key []byte) error            { panic("implement me") }
func (appliedTxsMock) Get(key []byte) ([]byte, error)     { panic("implement me") }
func (appliedTxsMock) Has(key []byte) (bool, error)       { panic("implement me") }
func (appliedTxsMock) Close()                             { panic("implement me") }
func (appliedTxsMock) NewBatch() database.Batch           { panic("implement me") }
func (appliedTxsMock) Find(key []byte) database.Iterator  { panic("implement me") }

func createTransaction(t *testing.T, nonce uint64, destination types.Address, amount, fee uint64, signer *signing.EdSigner) *types.Transaction {
	tx, err := transaction.GenerateCallTransaction(signer, destination, nonce, amount, 100, fee)
	assert.NoError(t, err)
	return tx
}

func newTx(t *testing.T, nonce, totalAmount uint64, signer *signing.EdSigner) *types.Transaction {
	feeAmount := uint64(1)
	rec := types.Address{byte(rand.Int()), byte(rand.Int()), byte(rand.Int()), byte(rand.Int())}
	return createTransaction(t, nonce, rec, totalAmount-feeAmount, feeAmount, signer)
}

func TestHandleGossipTransaction_ValidationAccepted(t *testing.T) {
	r := require.New(t)

	db := database.NewMemDatabase()
	lg := logtest.New(t).WithName("svm_logger")
	svm := New(db, appliedTxsMock{}, &ProjectorMock{}, mempool.NewTxMemPool(), lg)

	signer := signing.NewEdSigner()
	origin := types.GenerateAddress(signer.PublicKey().Bytes())
	svm.state.SetBalance(origin, 500)
	svm.state.SetNonce(origin, 3)

	tx := newTx(t, 3, 10, signer)
	msg, _ := types.InterfaceToBytes(tx)

	got := svm.HandleGossipTransaction(context.TODO(), p2p.Peer(signer.PublicKey().String()), msg)
	want := pubsub.ValidationAccept
	r.Equal(got, want)
}

func TestHandleGossipTransaction_ValidationIgnored_WrongNonce(t *testing.T) {
	r := require.New(t)

	db := database.NewMemDatabase()
	lg := logtest.New(t).WithName("svm_logger")
	svm := New(db, appliedTxsMock{}, &ProjectorMock{}, mempool.NewTxMemPool(), lg)

	signer := signing.NewEdSigner()
	origin := types.BytesToAddress(signer.PublicKey().Bytes())
	svm.state.SetBalance(origin, 500)
	svm.state.SetNonce(origin, 3)

	tx := newTx(t, 4, 10, signer)
	msg, _ := types.InterfaceToBytes(tx)

	got := svm.HandleGossipTransaction(context.TODO(), p2p.Peer(signer.PublicKey().String()), msg)
	want := pubsub.ValidationIgnore
	r.Equal(got, want)
}

func TestHandleGossipTransaction_ValidationIgnored_InsufficientBalance(t *testing.T) {
	r := require.New(t)

	db := database.NewMemDatabase()
	lg := logtest.New(t).WithName("svm_logger")
	svm := New(db, appliedTxsMock{}, &ProjectorMock{}, mempool.NewTxMemPool(), lg)

	signer := signing.NewEdSigner()
	origin := types.BytesToAddress(signer.PublicKey().Bytes())
	svm.state.SetBalance(origin, 5)
	svm.state.SetNonce(origin, 3)

	tx := newTx(t, 3, 10, signer)
	msg, _ := types.InterfaceToBytes(tx)

	got := svm.HandleGossipTransaction(context.TODO(), p2p.Peer(signer.PublicKey().String()), msg)
	want := pubsub.ValidationIgnore
	r.Equal(got, want)
}

func TestHandleGossipTransaction_ValidationIgnored_NoTxOrigin(t *testing.T) {
	r := require.New(t)

	db := database.NewMemDatabase()
	lg := logtest.New(t).WithName("svm_logger")
	svm := New(db, appliedTxsMock{}, &ProjectorMock{}, mempool.NewTxMemPool(), lg)

	signer := signing.NewEdSigner()
	tx := newTx(t, 3, 10, signer)
	msg, _ := types.InterfaceToBytes(tx)

	got := svm.HandleGossipTransaction(context.TODO(), p2p.Peer(signer.PublicKey().String()), msg)
	want := pubsub.ValidationIgnore
	r.Equal(got, want)
}
