package activation

import (
	"github.com/google/uuid"
	"github.com/spacemeshos/go-spacemesh/common"
	"github.com/spacemeshos/go-spacemesh/database"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/mesh"
	"github.com/spacemeshos/go-spacemesh/nipst"
	"github.com/spacemeshos/go-spacemesh/types"
	"github.com/stretchr/testify/assert"
	"math/big"
	"strconv"
	"testing"
)

func createLayerWithAtx(msh *mesh.Mesh, atxdb *ActivationDb, id types.LayerID, numOfBlocks int, atxs []*types.ActivationTx, votes []types.BlockID, views []types.BlockID) (created []types.BlockID) {
	for i := 0; i < numOfBlocks; i++ {
		block1 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), id, []byte("data1"))
		block1.MinerID.Key = strconv.Itoa(i)
		block1.ATXs = append(block1.ATXs, atxs...)
		block1.BlockVotes = append(block1.BlockVotes, votes...)
		block1.ViewEdges = append(block1.ViewEdges, views...)
		msh.AddBlock(block1)
		created = append(created, block1.Id)
		for _, atx := range block1.ATXs {
			atxdb.StoreAtx(atx.PubLayerIdx.GetEpoch(1000), atx)
		}
	}
	return
}

type MeshValidatorMock struct{}

func (m *MeshValidatorMock) HandleIncomingLayer(layer *types.Layer) (types.LayerID, types.LayerID) {
	return layer.Index() - 1, layer.Index()
}
func (m *MeshValidatorMock) HandleLateBlock(bl *types.Block)              {}
func (m *MeshValidatorMock) RegisterLayerCallback(func(id types.LayerID)) {}
func (mlg *MeshValidatorMock) ContextualValidity(id types.BlockID) bool   { return true }

type MockState struct{}

func (MockState) ApplyTransactions(layer types.LayerID, txs mesh.Transactions) (uint32, error) {
	return 0, nil
}

func (MockState) ApplyRewards(layer types.LayerID, miners []string, underQuota map[string]int, bonusReward, diminishedReward *big.Int) {
}

type AtxDbMock struct{}

func (AtxDbMock) ProcessBlockATXs(block *types.Block) {

}

func ConfigTst() mesh.Config {
	return mesh.Config{
		big.NewInt(10),
		big.NewInt(5000),
		big.NewInt(15),
		15,
		5,
	}
}

func getAtxDb(id string) (*ActivationDb, *mesh.Mesh) {
	lg := log.NewDefault(id)
	memesh := mesh.NewMemMeshDB(lg.WithName("meshDB"))
	atxdb := NewActivationDb(database.NewMemDatabase(), database.NewMemDatabase(), NewIdentityStore(database.NewMemDatabase()), memesh, 1000, &ValidatorMock{}, lg.WithName("atxDB"))
	layers := mesh.NewMesh(memesh, atxdb, ConfigTst(), &MeshValidatorMock{}, &MockState{}, lg.WithName("mesh"))
	return atxdb, layers
}

func Test_CalcActiveSetFromView(t *testing.T) {
	activesetCache.Purge()
	atxdb, layers := getAtxDb("t6")

	id1 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	id2 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	id3 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	atxs := []*types.ActivationTx{
		types.NewActivationTx(id1, 0, *types.EmptyAtxId, 12, 0, *types.EmptyAtxId, 0, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id2, 0, *types.EmptyAtxId, 300, 0, *types.EmptyAtxId, 0, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id3, 0, *types.EmptyAtxId, 435, 0, *types.EmptyAtxId, 0, []types.BlockID{}, &nipst.NIPST{}, true),
	}

	for _, atx := range atxs {
		hash, err := atx.NIPSTChallenge.Hash()
		assert.NoError(t, err)
		atx.Nipst = nipst.NewNIPSTWithChallenge(hash)
	}

	blocks := createLayerWithAtx(layers, atxdb, 1, 10, atxs, []types.BlockID{}, []types.BlockID{})
	blocks = createLayerWithAtx(layers, atxdb, 10, 10, []*types.ActivationTx{}, blocks, blocks)
	blocks = createLayerWithAtx(layers, atxdb, 100, 10, []*types.ActivationTx{}, blocks, blocks)

	atx := types.NewActivationTx(id1, 1, atxs[0].Id(), 1000, 0, atxs[0].Id(), 3, blocks, &nipst.NIPST{}, true)
	num, err := atxdb.CalcActiveSetFromView(atx)
	assert.NoError(t, err)
	assert.Equal(t, 3, int(num))

	// check that further atxs dont affect current epoch count
	atxs2 := []*types.ActivationTx{
		types.NewActivationTx(types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}, 0, *types.EmptyAtxId, 1012, 0, atxs[0].Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}, 0, *types.EmptyAtxId, 1300, 0, atxs[1].Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}, 0, *types.EmptyAtxId, 1435, 0, atxs[2].Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true),
	}

	for _, atx := range atxs2 {
		hash, err := atx.NIPSTChallenge.Hash()
		assert.NoError(t, err)
		atx.Nipst = nipst.NewNIPSTWithChallenge(hash)
	}

	block2 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 2200, []byte("data1"))
	block2.MinerID.Key = strconv.Itoa(1)
	block2.ATXs = append(block2.ATXs, atxs2...)
	block2.ViewEdges = blocks
	layers.AddBlock(block2)
	atxdb.ProcessBlockATXs(block2)

	atx2 := types.NewActivationTx(id3, 0, *types.EmptyAtxId, 1435, 0, *types.EmptyAtxId, 6, []types.BlockID{block2.Id}, &nipst.NIPST{}, true)
	num, err = atxdb.CalcActiveSetFromView(atx2)
	assert.NoError(t, err)
	assert.Equal(t, 3, int(num))
}

func Test_DBSanity(t *testing.T) {
	atxdb, _ := getAtxDb("t6")

	id1 := types.NodeId{Key: uuid.New().String()}
	id2 := types.NodeId{Key: uuid.New().String()}
	id3 := types.NodeId{Key: uuid.New().String()}

	atx1 := types.NewActivationTx(id1, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true)
	atx2 := types.NewActivationTx(id1, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true)
	atx3 := types.NewActivationTx(id1, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true)

	err := atxdb.storeAtxUnlocked(atx1)
	assert.NoError(t, err)
	err = atxdb.storeAtxUnlocked(atx2)
	assert.NoError(t, err)
	err = atxdb.storeAtxUnlocked(atx3)
	assert.NoError(t, err)

	err = atxdb.addAtxToNodeIdSorted(id1, atx1)
	assert.NoError(t, err)
	ids, err := atxdb.GetNodeAtxIds(id1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ids))
	assert.Equal(t, atx1.Id(), ids[0])

	err = atxdb.addAtxToNodeIdSorted(id2, atx2)
	assert.NoError(t, err)

	err = atxdb.addAtxToNodeIdSorted(id1, atx3)
	assert.NoError(t, err)

	ids, err = atxdb.GetNodeAtxIds(id2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ids))
	assert.Equal(t, atx2.Id(), ids[0])

	ids, err = atxdb.GetNodeAtxIds(id1)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ids))
	assert.Equal(t, atx1.Id(), ids[0])

	ids, err = atxdb.GetNodeAtxIds(id3)
	assert.Error(t, err)
	assert.Equal(t, 0, len(ids))
}

func Test_Wrong_CalcActiveSetFromView(t *testing.T) {
	atxdb, layers := getAtxDb("t6")

	id1 := types.NodeId{Key: uuid.New().String()}
	id2 := types.NodeId{Key: uuid.New().String()}
	id3 := types.NodeId{Key: uuid.New().String()}
	atxs := []*types.ActivationTx{
		types.NewActivationTx(id1, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id2, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id3, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
	}

	blocks := createLayerWithAtx(layers, atxdb, 1, 10, atxs, []types.BlockID{}, []types.BlockID{})
	blocks = createLayerWithAtx(layers, atxdb, 10, 10, []*types.ActivationTx{}, blocks, blocks)
	blocks = createLayerWithAtx(layers, atxdb, 100, 10, []*types.ActivationTx{}, blocks, blocks)

	atx := types.NewActivationTx(id1, 1, atxs[0].Id(), 1000, 0, atxs[0].Id(), 20, blocks, &nipst.NIPST{}, true)
	num, err := atxdb.CalcActiveSetFromView(atx)
	assert.NoError(t, err)
	assert.NotEqual(t, 20, int(num))

}

func TestMesh_processBlockATXs(t *testing.T) {
	activesetCache.Purge()
	atxdb, _ := getAtxDb("t6")

	id1 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	id2 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	id3 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	chlng := common.HexToHash("0x3333")
	npst := nipst.NewNIPSTWithChallenge(&chlng)
	posATX := types.NewActivationTx(types.NodeId{"aaaaaa", []byte("anton")}, 0, *types.EmptyAtxId, 1000, 0, *types.EmptyAtxId, 0, []types.BlockID{}, npst, true)
	err := atxdb.StoreAtx(0, posATX)
	assert.NoError(t, err)
	atxs := []*types.ActivationTx{
		types.NewActivationTx(id1, 0, *types.EmptyAtxId, 1012, 0, posATX.Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id2, 0, *types.EmptyAtxId, 1300, 0, posATX.Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id3, 0, *types.EmptyAtxId, 1435, 0, posATX.Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true),
	}
	for _, atx := range atxs {
		hash, err := atx.NIPSTChallenge.Hash()
		assert.NoError(t, err)
		atx.Nipst = nipst.NewNIPSTWithChallenge(hash)
	}

	block1 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 1, []byte("data1"))
	block1.MinerID.Key = strconv.Itoa(1)
	block1.ATXs = append(block1.ATXs, atxs...)

	atxdb.ProcessBlockATXs(block1)
	assert.Equal(t, 3, int(atxdb.ActiveSetSize(1)))

	// check that further atxs dont affect current epoch count
	atxs2 := []*types.ActivationTx{
		types.NewActivationTx(id1, 1, atxs[0].Id(), 2012, 0, atxs[0].Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id2, 1, atxs[1].Id(), 2300, 0, atxs[1].Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id3, 1, atxs[2].Id(), 2435, 0, atxs[2].Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true),
	}
	for _, atx := range atxs2 {
		hash, err := atx.NIPSTChallenge.Hash()
		assert.NoError(t, err)
		atx.Nipst = nipst.NewNIPSTWithChallenge(hash)
	}

	block2 := types.NewExistingBlock(types.BlockID(uuid.New().ID()), 2000, []byte("data1"))
	block2.MinerID.Key = strconv.Itoa(1)
	block2.ATXs = append(block2.ATXs, atxs2...)
	atxdb.ProcessBlockATXs(block2)

	assert.Equal(t, 3, int(atxdb.ActiveSetSize(1)))
	assert.Equal(t, 3, int(atxdb.ActiveSetSize(2)))
}

func TestActivationDB_ValidateAtx(t *testing.T) {
	atxdb, layers := getAtxDb("t8")

	idx1 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}

	id1 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	id2 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	id3 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	atxs := []*types.ActivationTx{
		types.NewActivationTx(id1, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id2, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id3, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
	}
	for _, atx := range atxs {
		hash, err := atx.NIPSTChallenge.Hash()
		assert.NoError(t, err)
		atx.Nipst = nipst.NewNIPSTWithChallenge(hash)
	}

	blocks := createLayerWithAtx(layers, atxdb, 1, 10, atxs, []types.BlockID{}, []types.BlockID{})
	blocks = createLayerWithAtx(layers, atxdb, 10, 10, []*types.ActivationTx{}, blocks, blocks)
	blocks = createLayerWithAtx(layers, atxdb, 100, 10, []*types.ActivationTx{}, blocks, blocks)

	//atx := types.NewActivationTx(id1, 1, atxs[0].Id(), 1000, 0, atxs[0].Id(), 3, blocks, &nipst.NIPST{})
	prevAtx := types.NewActivationTx(idx1, 0, *types.EmptyAtxId, 100, 0, *types.EmptyAtxId, 3, blocks, &nipst.NIPST{}, true)
	prevAtx.Valid = true
	hash, err := prevAtx.NIPSTChallenge.Hash()
	assert.NoError(t, err)
	prevAtx.Nipst = nipst.NewNIPSTWithChallenge(hash)

	atx := types.NewActivationTx(idx1, 1, prevAtx.Id(), 1012, 0, prevAtx.Id(), 3, []types.BlockID{}, &nipst.NIPST{}, true)
	atx.VerifiedActiveSet = 3
	hash, err = atx.NIPSTChallenge.Hash()
	assert.NoError(t, err)
	atx.Nipst = nipst.NewNIPSTWithChallenge(hash)
	err = atxdb.StoreAtx(1, prevAtx)
	assert.NoError(t, err)

	err = atxdb.ValidateAtx(atx)
	assert.NoError(t, err)
}

func TestActivationDB_ValidateAtxErrors(t *testing.T) {
	atxdb, layers := getAtxDb("t8")

	idx1 := types.NodeId{Key: uuid.New().String()}

	id1 := types.NodeId{Key: uuid.New().String()}
	id2 := types.NodeId{Key: uuid.New().String()}
	id3 := types.NodeId{Key: uuid.New().String()}
	atxs := []*types.ActivationTx{
		types.NewActivationTx(id1, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id2, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id3, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
	}

	blocks := createLayerWithAtx(layers, atxdb, 1, 10, atxs, []types.BlockID{}, []types.BlockID{})
	blocks = createLayerWithAtx(layers, atxdb, 10, 10, []*types.ActivationTx{}, blocks, blocks)
	blocks = createLayerWithAtx(layers, atxdb, 100, 10, []*types.ActivationTx{}, blocks, blocks)

	//atx := types.NewActivationTx(id1, 1, atxs[0].Id(), 1000, 0, atxs[0].Id(), 3, blocks, &nipst.NIPST{})
	chlng := common.HexToHash("0x3333")
	npst := nipst.NewNIPSTWithChallenge(&chlng)
	prevAtx := types.NewActivationTx(idx1, 0, *types.EmptyAtxId, 100, 0, *types.EmptyAtxId, 3, blocks, npst, true)
	prevAtx.Valid = true

	err := atxdb.StoreAtx(1, prevAtx)
	assert.NoError(t, err)

	//todo: can test against exact error strings
	//wrong sequnce
	atx := types.NewActivationTx(idx1, 0, prevAtx.Id(), 1012, 0, prevAtx.Id(), 3, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.ValidateAtx(atx)
	assert.Error(t, err)

	//wrong active set
	atx = types.NewActivationTx(idx1, 1, prevAtx.Id(), 1012, 0, prevAtx.Id(), 10, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.ValidateAtx(atx)
	assert.Error(t, err)

	//wrong positioning atx
	atx = types.NewActivationTx(idx1, 1, prevAtx.Id(), 1012, 0, atxs[0].Id(), 3, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.ValidateAtx(atx)
	assert.Error(t, err)

	//wrong prevATx
	atx = types.NewActivationTx(idx1, 1, atxs[0].Id(), 1012, 0, prevAtx.Id(), 3, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.ValidateAtx(atx)
	assert.Error(t, err)

	//wrong layerId
	atx = types.NewActivationTx(idx1, 1, prevAtx.Id(), 12, 0, prevAtx.Id(), 3, []types.BlockID{}, npst, true)
	err = atxdb.ValidateAtx(atx)
	assert.Error(t, err)

	//atx already exists
	err = atxdb.StoreAtx(1, atx)
	assert.NoError(t, err)
	atx = types.NewActivationTx(idx1, 1, prevAtx.Id(), 12, 0, prevAtx.Id(), 3, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.ValidateAtx(atx)
	assert.Error(t, err)
	//atx = types.NewActivationTx(idx1, 1, prevAtx.Id(), 1012, 0, prevAtx.Id(), 3, []types.BlockID{}, &nipst.NIPST{})
}

func TestActivationDB_ValidateAndInsertSorted(t *testing.T) {
	atxdb, layers := getAtxDb("t8")

	idx1 := types.NodeId{Key: uuid.New().String()}

	id1 := types.NodeId{Key: uuid.New().String()}
	id2 := types.NodeId{Key: uuid.New().String()}
	id3 := types.NodeId{Key: uuid.New().String()}
	atxs := []*types.ActivationTx{
		types.NewActivationTx(id1, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id2, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
		types.NewActivationTx(id3, 0, *types.EmptyAtxId, 1, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true),
	}

	blocks := createLayerWithAtx(layers, atxdb, 1, 10, atxs, []types.BlockID{}, []types.BlockID{})
	blocks = createLayerWithAtx(layers, atxdb, 10, 10, []*types.ActivationTx{}, blocks, blocks)
	blocks = createLayerWithAtx(layers, atxdb, 100, 10, []*types.ActivationTx{}, blocks, blocks)

	//atx := types.NewActivationTx(id1, 1, atxs[0].Id(), 1000, 0, atxs[0].Id(), 3, blocks, &nipst.NIPST{})
	chlng := common.HexToHash("0x3333")
	npst := nipst.NewNIPSTWithChallenge(&chlng)
	prevAtx := types.NewActivationTx(idx1, 0, *types.EmptyAtxId, 100, 0, *types.EmptyAtxId, 3, blocks, npst, true)
	prevAtx.Valid = true

	var nodeAtxIds []types.AtxId

	err := atxdb.StoreAtx(1, prevAtx)
	assert.NoError(t, err)
	nodeAtxIds = append(nodeAtxIds, prevAtx.Id())

	//wrong sequnce
	atx := types.NewActivationTx(idx1, 1, prevAtx.Id(), 1012, 0, prevAtx.Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.StoreAtx(1, atx)
	assert.NoError(t, err)
	nodeAtxIds = append(nodeAtxIds, atx.Id())

	atx = types.NewActivationTx(idx1, 2, atx.Id(), 1012, 0, atx.Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.StoreAtx(1, atx)
	assert.NoError(t, err)
	nodeAtxIds = append(nodeAtxIds, atx.Id())
	atx2id := atx.Id()

	atx = types.NewActivationTx(idx1, 4, atx.Id(), 1012, 0, prevAtx.Id(), 0, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.ValidateAtx(atx)
	assert.Error(t, err)
	assert.Equal(t, "sequence number is not one more than prev sequence number", err.Error())

	err = atxdb.StoreAtx(1, atx)
	assert.NoError(t, err)
	id4 := atx.Id()

	atx = types.NewActivationTx(idx1, 3, atx2id, 1012, 0, prevAtx.Id(), 3, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.ValidateAtx(atx)
	assert.Error(t, err)
	assert.Equal(t, "last atx is not the one referenced", err.Error())

	err = atxdb.StoreAtx(1, atx)
	assert.NoError(t, err)
	nodeAtxIds = append(nodeAtxIds, atx.Id())
	nodeAtxIds = append(nodeAtxIds, id4)

	ids, err := atxdb.GetNodeAtxIds(idx1)
	assert.True(t, len(ids) == 5)
	assert.Equal(t, ids, nodeAtxIds)

	_, err = atxdb.GetAtx(ids[len(ids)-1])
	assert.NoError(t, err)

	_, err = atxdb.GetAtx(ids[len(ids)-2])
	assert.NoError(t, err)

	//test same sequence
	idx2 := types.NodeId{Key: uuid.New().String()}

	prevAtx = types.NewActivationTx(idx2, 0, *types.EmptyAtxId, 100, 0, *types.EmptyAtxId, 3, blocks, npst, true)
	prevAtx.Valid = true
	err = atxdb.StoreAtx(1, prevAtx)
	assert.NoError(t, err)

	atx = types.NewActivationTx(idx2, 1, prevAtx.Id(), 1012, 0, prevAtx.Id(), 3, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.StoreAtx(1, atx)
	assert.NoError(t, err)
	atxId := atx.Id()

	atx = types.NewActivationTx(idx2, 2, atxId, 1012, 0, atx.Id(), 3, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.StoreAtx(1, atx)
	assert.NoError(t, err)

	atx = types.NewActivationTx(idx2, 2, atxId, 1013, 0, atx.Id(), 3, []types.BlockID{}, &nipst.NIPST{}, true)
	err = atxdb.ValidateAtx(atx)
	assert.Error(t, err)
	assert.Equal(t, "last atx is not the one referenced", err.Error())

	err = atxdb.StoreAtx(1, atx)
	assert.NoError(t, err)

}

func TestActivationDb_ProcessAtx(t *testing.T) {
	atxdb, _ := getAtxDb("t8")
	idx1 := types.NodeId{Key: uuid.New().String(), VRFPublicKey: []byte("anton")}
	atx := types.NewActivationTx(idx1, 0, *types.EmptyAtxId, 100, 0, *types.EmptyAtxId, 3, []types.BlockID{}, &nipst.NIPST{}, true)
	atxdb.ProcessAtx(atx)
	res, err := atxdb.ids.GetIdentity(idx1.Key)
	assert.Nil(t, err)
	assert.Equal(t, idx1, res)
}