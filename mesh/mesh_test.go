package mesh

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"github.com/spacemeshos/go-spacemesh/address"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/nipst"
	"github.com/spacemeshos/go-spacemesh/rand"
	"github.com/spacemeshos/go-spacemesh/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/big"
	"strconv"
	"testing"
	"time"
)

type MeshValidatorMock struct{}

func (m *MeshValidatorMock) HandleIncomingLayer(layer *types.Layer) (types.LayerID, types.LayerID) {
	return layer.Index() - 1, layer.Index()
}
func (m *MeshValidatorMock) HandleLateBlock(bl *types.Block)              {}
func (m *MeshValidatorMock) RegisterLayerCallback(func(id types.LayerID)) {}
func (mlg *MeshValidatorMock) ContextualValidity(id types.BlockID) bool   { return true }

type MockState struct{}

func (MockState) ValidateSignature(signed types.Signed) (address.Address, error) {
	return address.Address{}, nil
}

func (MockState) ApplyTransactions(layer types.LayerID, txs Transactions) (uint32, error) {
	return 0, nil
}

func (MockState) ApplyRewards(layer types.LayerID, miners []address.Address, underQuota map[address.Address]int, bonusReward, diminishedReward *big.Int) {
}

func (MockState) ValidateTransactionSignature(tx *types.SerializableSignedTransaction) (address.Address, error) {
	return address.Address{}, nil
}

type AtxDbMock struct {
	db     map[types.AtxId]*types.ActivationTx
	nipsts map[types.AtxId]*types.NIPST
}

func (*AtxDbMock) IsIdentityActive(edId string, layer types.LayerID) (bool, types.AtxId, error) {
	return true, *types.EmptyAtxId, nil
}
func (t *AtxDbMock) GetEpochAtxIds(id types.EpochId) ([]types.AtxId, error) {
	return []types.AtxId{}, nil /*todo: mock if needed */
}

func (t *AtxDbMock) GetAtx(id types.AtxId) (*types.ActivationTx, error) {
	if id == *types.EmptyAtxId {
		return nil, fmt.Errorf("trying to fetch empty atx id")
	}

	if atx, ok := t.db[id]; ok {
		return atx, nil
	}
	return nil, fmt.Errorf("cannot find atx")
}

func (t *AtxDbMock) AddAtx(id types.AtxId, atx *types.ActivationTx) {
	t.db[id] = atx
	t.nipsts[id] = atx.Nipst
}

func (t *AtxDbMock) GetNipst(id types.AtxId) (*types.NIPST, error) {
	return t.nipsts[id], nil
}

func (AtxDbMock) ProcessAtx(atx *types.ActivationTx) {

}

func (AtxDbMock) SyntacticallyValidateAtx(atx *types.ActivationTx) error {
	return nil
}

type MockTxMemPool struct{}

func (MockTxMemPool) Get(id types.TransactionId) (types.AddressableSignedTransaction, error) {
	return types.AddressableSignedTransaction{}, nil
}
func (MockTxMemPool) PopItems(size int) []types.AddressableSignedTransaction {
	return nil
}
func (MockTxMemPool) Put(id types.TransactionId, item *types.AddressableSignedTransaction) {

}
func (MockTxMemPool) Invalidate(id types.TransactionId) {

}

type MockAtxMemPool struct{}

func (MockAtxMemPool) Get(id types.AtxId) (types.ActivationTx, error) {
	return types.ActivationTx{}, nil
}

func (MockAtxMemPool) PopItems(size int) []types.ActivationTx {
	return nil
}

func (MockAtxMemPool) Put(id types.AtxId, item *types.ActivationTx) {

}

func (MockAtxMemPool) Invalidate(id types.AtxId) {

}

func getMesh(id string) *Mesh {
	lg := log.New(id, "", "")
	layers := NewMesh(NewMemMeshDB(lg), &AtxDbMock{}, ConfigTst(), &MeshValidatorMock{}, MockTxMemPool{}, MockAtxMemPool{}, &MockState{}, lg)
	return layers
}

func TestLayers_AddBlock(t *testing.T) {

	layers := getMesh("t1")
	defer layers.Close()

	block1 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 1, []byte("data1"))
	block2 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 2, []byte("data2"))
	block3 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 3, []byte("data3"))

	addTransactionsWithGas(layers.MeshDB, block1, 4, rand.Int63n(100))

	err := layers.AddBlock(block1)
	assert.NoError(t, err)
	err = layers.AddBlock(block2)
	assert.NoError(t, err)
	err = layers.AddBlock(block3)
	assert.NoError(t, err)

	rBlock2, err := layers.GetBlock(block2.Id)
	assert.NoError(t, err)

	rBlock1, err := layers.GetBlock(block1.Id)
	assert.NoError(t, err)

	assert.True(t, len(rBlock1.TxIds) == len(block1.TxIds), "block content was wrong")
	assert.True(t, bytes.Compare(rBlock2.MiniBlock.Data, []byte("data2")) == 0, "block content was wrong")
	assert.True(t, len(rBlock1.AtxIds) == len(block1.AtxIds))
}

func TestLayers_AddLayer(t *testing.T) {
	layers := getMesh("t2")
	defer layers.Close()
	id := types.LayerID(1)
	block1 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), id, []byte("data"))
	block2 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), id, []byte("data"))
	block3 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), id, []byte("data"))
	l, err := layers.GetLayer(id)
	assert.True(t, err != nil, "error: ", err)

	err = layers.AddBlock(block1)
	assert.NoError(t, err)
	err = layers.AddBlock(block2)
	assert.NoError(t, err)
	err = layers.AddBlock(block3)
	assert.NoError(t, err)
	l, err = layers.GetLayer(id)
	assert.NoError(t, err)
	//assert.True(t, layers.VerifiedLayer() == 0, "wrong layer count")
	assert.True(t, string(l.Blocks()[1].MiniBlock.Data) == "data", "wrong block data ")
}

func TestLayers_AddWrongLayer(t *testing.T) {
	layers := getMesh("t3")
	defer layers.Close()
	block1 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 1, []byte("data data data"))
	block2 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 2, []byte("data data data"))
	block3 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 4, []byte("data data data"))
	l1 := types.NewExistingLayer(1, []*types.Block{block1})
	layers.AddBlock(block1)
	layers.ValidateLayer(l1)
	l2 := types.NewExistingLayer(2, []*types.Block{block2})
	layers.AddBlock(block2)
	layers.ValidateLayer(l2)
	layers.AddBlock(block3)
	_, err := layers.GetVerifiedLayer(1)
	assert.True(t, err == nil, "error: ", err)
	_, err1 := layers.GetVerifiedLayer(2)
	assert.True(t, err1 == nil, "error: ", err1)
	_, err2 := layers.GetVerifiedLayer(4)
	assert.True(t, err2 != nil, "added wrong layer ", err2)
}

func TestLayers_GetLayer(t *testing.T) {
	layers := getMesh("t4")
	defer layers.Close()
	block1 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 1, []byte("data data data"))
	block2 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 1, []byte("data data data"))
	block3 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 1, []byte("data data data"))
	l1 := types.NewExistingLayer(1, []*types.Block{block1})
	layers.AddBlock(block1)
	layers.ValidateLayer(l1)
	l, err := layers.GetVerifiedLayer(0)
	layers.AddBlock(block2)
	layers.AddBlock(block3)
	l, err = layers.GetVerifiedLayer(1)
	assert.True(t, err == nil, "error: ", err)
	assert.True(t, l.Index() == 1, "wrong layer")
}

func TestLayers_LatestKnownLayer(t *testing.T) {
	layers := getMesh("t6")
	defer layers.Close()
	layers.SetLatestLayer(3)
	layers.SetLatestLayer(7)
	layers.SetLatestLayer(10)
	layers.SetLatestLayer(1)
	layers.SetLatestLayer(2)
	assert.True(t, layers.LatestLayer() == 10, "wrong layer")
}

func TestLayers_WakeUp(t *testing.T) {
	//layers := getMesh(make(chan Peer),  "t5")
	//defer layers.Close()
	//layers.SetLatestLayer(10)
	//assert.True(t, layers.LocalLayerCount() == 10, "wrong layer")
}

func TestLayers_OrphanBlocks(t *testing.T) {
	layers := getMesh("t6")
	defer layers.Close()
	block1 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 1, []byte("data data data"))
	block2 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 1, []byte("data data data"))
	block3 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 2, []byte("data data data"))
	block4 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 2, []byte("data data data"))
	block5 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 3, []byte("data data data"))
	block5.AddView(block1.ID())
	block5.AddView(block2.ID())
	block5.AddView(block3.ID())
	block5.AddView(block4.ID())
	layers.AddBlock(block1)
	layers.AddBlock(block2)
	layers.AddBlock(block3)
	layers.AddBlock(block4)
	arr, _ := layers.GetOrphanBlocksBefore(3)
	assert.True(t, len(arr) == 4, "wrong layer")
	arr2, _ := layers.GetOrphanBlocksBefore(2)
	assert.Equal(t, len(arr2), 2)
	layers.AddBlock(block5)
	time.Sleep(1 * time.Second)
	arr3, _ := layers.GetOrphanBlocksBefore(4)
	assert.True(t, len(arr3) == 1, "wrong layer")

}

func createLayerWithAtx(t *testing.T, msh *Mesh, id types.LayerID, numOfBlocks int, atxs []*types.ActivationTx, votes []types.BlockID, views []types.BlockID) (created []types.BlockID) {
	for i := 0; i < numOfBlocks; i++ {
		block1 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), id, []byte("data1"))
		block1.MinerID.Key = strconv.Itoa(i)
		block1.BlockVotes = append(block1.BlockVotes, votes...)
		for _, atx := range atxs {
			block1.AtxIds = append(block1.AtxIds, atx.Id())
		}
		block1.ViewEdges = append(block1.ViewEdges, views...)
		err := msh.AddBlockWithTxs(block1, []*types.AddressableSignedTransaction{}, atxs)
		require.NoError(t, err)
		created = append(created, block1.Id)
	}
	return
}

func TestMesh_ActiveSetForLayerView(t *testing.T) {
	layers := getMesh(t.Name())
	layers.AtxDB = &AtxDbMock{make(map[types.AtxId]*types.ActivationTx), make(map[types.AtxId]*types.NIPST)}

	id1 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	id2 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	id3 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	coinbase1 := address.HexToAddress("aaaa")
	coinbase2 := address.HexToAddress("bbbb")
	coinbase3 := address.HexToAddress("cccc")
	atxs := []*types.ActivationTx{
		types.NewActivationTx(id1, coinbase1, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 0, []types.BlockID{}, &types.NIPST{}),
		types.NewActivationTx(id1, coinbase1, 0, *types.EmptyAtxId, 2, 0, *types.EmptyAtxId, 0, []types.BlockID{}, &types.NIPST{}),
		types.NewActivationTx(id1, coinbase1, 0, *types.EmptyAtxId, 3, 0, *types.EmptyAtxId, 0, []types.BlockID{}, &types.NIPST{}),
		types.NewActivationTx(id2, coinbase2, 0, *types.EmptyAtxId, 2, 0, *types.EmptyAtxId, 0, []types.BlockID{}, &types.NIPST{}),
		types.NewActivationTx(id3, coinbase3, 0, *types.EmptyAtxId, 11, 0, *types.EmptyAtxId, 0, []types.BlockID{}, &types.NIPST{}),
	}

	poetRef := []byte{0xba, 0xb0}
	for _, atx := range atxs {
		hash, err := atx.NIPSTChallenge.Hash()
		assert.NoError(t, err)
		atx.Nipst = nipst.NewNIPSTWithChallenge(hash, poetRef)
		layers.AtxDB.(*AtxDbMock).AddAtx(atx.Id(), atx)
	}

	blocks := createLayerWithAtx(t, layers, 1, 1, atxs, []types.BlockID{}, []types.BlockID{})
	for i := 2; i <= 10; i++ {
		blocks = createLayerWithAtx(t, layers, types.LayerID(i), 1, []*types.ActivationTx{}, blocks, blocks)

	}

	actives, err := layers.ActiveSetForLayerConsensusView(10, 6)
	assert.NoError(t, err)
	assert.Equal(t, 1, int(len(actives)))
	_, ok := actives[id2.Key]
	assert.True(t, ok)
}
