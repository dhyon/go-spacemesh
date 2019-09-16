package miner

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/spacemeshos/ed25519"
	"github.com/spacemeshos/go-spacemesh/activation"
	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/common/util"
	"github.com/spacemeshos/go-spacemesh/config"
	"github.com/spacemeshos/go-spacemesh/events"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/mesh"
	"github.com/spacemeshos/go-spacemesh/oracle"
	"github.com/spacemeshos/go-spacemesh/p2p"
	"github.com/spacemeshos/go-spacemesh/p2p/service"
	"github.com/spacemeshos/go-spacemesh/signing"
	"math/rand"
	"sync"
	"time"
)

const MaxTransactionsPerBlock = 200 //todo: move to config
const MaxAtxPerBlock = 200          //todo: move to config

const DefaultGasLimit = 10
const DefaultGas = 1

const IncomingTxProtocol = "TxGossip"

const AtxsPerBlockLimit = 100

type Signer interface {
	Sign(m []byte) []byte
}

type TxValidator interface {
	GetValidAddressableTx(tx *types.SerializableSignedTransaction) (*types.AddressableSignedTransaction, error)
}

type AtxValidator interface {
	SyntacticallyValidateAtx(atx *types.ActivationTx) error
	ValidateSignedAtx(signedAtx *types.SignedAtx) error
	GetIdentity(id string) (types.NodeId, error)
}

type Syncer interface {
	FetchPoetProof(poetProofRef []byte) error
	WeaklySynced() bool
}

type BlockBuilder struct {
	log.Log
	Signer
	minerID          types.NodeId
	rnd              *rand.Rand
	hdist            types.LayerID
	beginRoundEvent  chan types.LayerID
	stopChan         chan struct{}
	txGossipChannel  chan service.GossipMessage
	atxGossipChannel chan service.GossipMessage
	hareResult       HareResultProvider
	AtxPool          *TypesAtxIdMemPool
	TransactionPool  *TypesTransactionIdMemPool
	mu               sync.Mutex
	network          p2p.Service
	weakCoinToss     WeakCoinProvider
	orphans          OrphanBlockProvider
	blockOracle      oracle.BlockOracle
	txValidator      TxValidator
	atxValidator     AtxValidator
	syncer           Syncer
	started          bool
	atxsPerBlock     int // number of atxs to select per block
}

func NewBlockBuilder(minerID types.NodeId, sgn Signer, net p2p.Service,
	beginRoundEvent chan types.LayerID, hdist int,
	txPool *TypesTransactionIdMemPool,
	atxPool *TypesAtxIdMemPool,
	weakCoin WeakCoinProvider,
	orph OrphanBlockProvider,
	hare HareResultProvider,
	blockOracle oracle.BlockOracle,
	txValidator TxValidator,
	atxValidator AtxValidator,
	syncer Syncer,
	atxsPerBlock int,
	lg log.Log) BlockBuilder {

	seed := binary.BigEndian.Uint64(md5.New().Sum([]byte(minerID.Key)))

	return BlockBuilder{
		minerID:          minerID,
		Signer:           sgn,
		hdist:            types.LayerID(hdist),
		Log:              lg,
		rnd:              rand.New(rand.NewSource(int64(seed))),
		beginRoundEvent:  beginRoundEvent,
		stopChan:         make(chan struct{}),
		AtxPool:          atxPool,
		TransactionPool:  txPool,
		txGossipChannel:  net.RegisterGossipProtocol(IncomingTxProtocol),
		atxGossipChannel: net.RegisterGossipProtocol(activation.AtxProtocol),
		hareResult:       hare,
		mu:               sync.Mutex{},
		network:          net,
		weakCoinToss:     weakCoin,
		orphans:          orph,
		blockOracle:      blockOracle,
		txValidator:      txValidator,
		atxValidator:     atxValidator,
		syncer:           syncer,
		started:          false,
		atxsPerBlock:     atxsPerBlock,
	}

}

func Transaction2SerializableTransaction(tx *mesh.Transaction) *types.AddressableSignedTransaction {
	inner := types.InnerSerializableSignedTransaction{
		AccountNonce: tx.AccountNonce,
		Recipient:    *tx.Recipient,
		Amount:       tx.Amount.Uint64(),
		GasLimit:     tx.GasLimit,
		GasPrice:     tx.GasPrice.Uint64(),
	}
	sst := &types.SerializableSignedTransaction{
		InnerSerializableSignedTransaction: inner,
	}
	return &types.AddressableSignedTransaction{
		SerializableSignedTransaction: sst,
		Address:                       tx.Origin,
	}
}

func (t *BlockBuilder) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.started {
		return fmt.Errorf("already started")
	}

	t.started = true
	go t.acceptBlockData()
	go t.listenForTx()
	go t.listenForAtx()
	return nil
}

func (t *BlockBuilder) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.started {
		return fmt.Errorf("already stopped")
	}
	t.started = false
	close(t.stopChan)
	return nil
}

type HareResultProvider interface {
	GetResult(lower types.LayerID, upper types.LayerID) ([]types.BlockID, error)
}

type WeakCoinProvider interface {
	GetResult() bool
}

type OrphanBlockProvider interface {
	GetOrphanBlocksBefore(l types.LayerID) ([]types.BlockID, error)
}

//used from external API call?
func (t *BlockBuilder) AddTransaction(tx *types.AddressableSignedTransaction) error {
	if !t.started {
		return fmt.Errorf("BlockBuilderStopped")
	}
	t.TransactionPool.Put(types.GetTransactionId(tx.SerializableSignedTransaction), tx)
	return nil
}

func calcHdistRange(id types.LayerID, hdist types.LayerID) (bottom types.LayerID, top types.LayerID) {
	if hdist == 0 {
		log.Panic("hdist cannot be zero")
	}

	bottom = types.LayerID(1)
	top = id - 1
	if id > hdist {
		bottom = id - hdist
	}

	return bottom, top
}

func (t *BlockBuilder) createBlock(id types.LayerID, atxID types.AtxId, eligibilityProof types.BlockEligibilityProof,
	txs []types.AddressableSignedTransaction, atxids []types.AtxId) (*types.Block, error) {

	var votes []types.BlockID = nil
	var err error
	if id == config.Genesis {
		return nil, errors.New("cannot create blockBytes in genesis layer")
	} else if id == config.Genesis+1 {
		votes = append(votes, config.GenesisId)
	} else { // get from hare
		bottom, top := calcHdistRange(id, t.hdist)
		votes, err = t.hareResult.GetResult(bottom, top)
		if err != nil {
			t.With().Warning("Could not get hare result during block creation",
				log.Uint64("bottom", uint64(bottom)), log.Uint64("top", uint64(top)),
				log.Uint64("hdist", uint64(t.hdist)), log.Err(err))
		}
		if votes == nil { // if no votes set to empty
			t.Info("Votes is nil. Setting votes to an empty array")
			votes = []types.BlockID{}
		}
	}

	viewEdges, err := t.orphans.GetOrphanBlocksBefore(id)
	if err != nil {
		return nil, err
	}

	var txids []types.TransactionId
	for _, t := range txs {
		txids = append(txids, types.GetTransactionId(t.SerializableSignedTransaction))
	}

	b := types.MiniBlock{
		BlockHeader: types.BlockHeader{
			Id:               types.BlockID(t.rnd.Int63()),
			LayerIndex:       id,
			ATXID:            atxID,
			EligibilityProof: eligibilityProof,
			Data:             nil,
			Coin:             t.weakCoinToss.GetResult(),
			Timestamp:        time.Now().UnixNano(),
			BlockVotes:       votes,
			ViewEdges:        viewEdges,
		},
		AtxIds: selectAtxs(atxids, t.atxsPerBlock),
		TxIds:  txids,
	}

	t.Log.Event().Info(fmt.Sprintf("I've created a block in layer %v. id: %v, num of transactions: %v, votes: %d, viewEdges: %d atx %v, atxs:%v",
		b.LayerIndex, b.ID(), len(b.TxIds), len(b.BlockVotes), len(b.ViewEdges), b.ATXID.ShortString(), len(b.AtxIds)))

	blockBytes, err := types.InterfaceToBytes(b)
	if err != nil {
		return nil, err
	}

	return &types.Block{MiniBlock: b, Signature: t.Signer.Sign(blockBytes)}, nil
}

func selectAtxs(atxs []types.AtxId, atxsPerBlock int) []types.AtxId {
	if len(atxs) == 0 { // no atxs to pick from
		return atxs
	}

	if len(atxs) <= atxsPerBlock { // no need to choose
		return atxs // take all
	}

	// we have more than atxsPerBlock, choose randomly
	selected := make([]types.AtxId, 0)
	for i := 0; i < atxsPerBlock; i++ {
		idx := i + rand.Intn(len(atxs)-i)       // random index in [i, len(atxs))
		selected = append(selected, atxs[idx])  // select atx at idx
		atxs[i], atxs[idx] = atxs[idx], atxs[i] // swap selected with i so we don't choose it again
	}

	return selected
}

func (t *BlockBuilder) listenForTx() {
	t.Log.Info("start listening for txs")
	for {
		select {
		case <-t.stopChan:
			return
		case data := <-t.txGossipChannel:
			if !t.syncer.WeaklySynced() {
				// not accepting txs when not synced
				continue
			}
			if data != nil {

				x, err := types.BytesAsSignedTransaction(data.Bytes())
				if err != nil {
					t.Log.Error("cannot parse incoming TX")
					continue
				}

				id := types.GetTransactionId(x)
				fullTx, err := t.txValidator.GetValidAddressableTx(x)
				if err != nil {
					t.Log.Error("Transaction validation failed for id=%v, err=%v", id, err)
					continue
				}

				t.Log.With().Info("got new tx", log.TxId(hex.EncodeToString(id[:util.Min(5, len(id))])))
				data.ReportValidation(IncomingTxProtocol)
				t.TransactionPool.Put(types.GetTransactionId(x), fullTx)
			}
		}
	}
}

func (t *BlockBuilder) listenForAtx() {
	t.Info("start listening for atxs")
	for {
		select {
		case <-t.stopChan:
			return
		case data := <-t.atxGossipChannel:
			if !t.syncer.WeaklySynced() {
				// not accepting atxs when not synced
				continue
			}
			t.handleGossipAtx(data)
		}
	}
}

// ValidateSignedAtx extracts public key from message and verifies public key exists in IdStore, this is how we validate
// ATX signature. If this is the first ATX it is considered valid anyways and ATX syntactic validation will determine ATX validity
func (t *BlockBuilder) ExtractPublicKey(signedAtx *types.SignedAtx) (*signing.PublicKey, error) {
	bts, err := signedAtx.AtxBytes()
	if err != nil {
		return nil, err
	}

	pubKey, err := ed25519.ExtractPublicKey(bts, signedAtx.Sig)
	if err != nil {
		return nil, err
	}

	pub := signing.NewPublicKey(pubKey)
	return pub, nil
}

func (t *BlockBuilder) handleGossipAtx(data service.GossipMessage) {
	if data == nil {
		return
	}
	signedAtx, err := types.BytesAsSignedAtx(data.Bytes())
	if err != nil {
		t.Error("cannot parse incoming ATX")
		return
	}

	/*err = t.atxValidator.ValidateSignedAtx(signedAtx)
	if err != nil {
		log.Error("cannot validate atx sig atx id %v err %v", signedAtx.Id(), err)
		return
	}*/
	pub, err := t.ExtractPublicKey(signedAtx)
	if err != nil {
		log.Error("cannot validate atx sig atx id %v err %v", signedAtx.Id(), err)
		return
	}

	if signedAtx.ActivationTx.PrevATXId != *types.EmptyAtxId {
		_, err = t.atxValidator.GetIdentity(pub.String())
		if err != nil { // means there is no such identity
			log.Error("no id found %v err %v", signedAtx.Id(), err)
			return
		}
	}

	atx := signedAtx.ActivationTx
	t.With().Info("got new ATX", log.AtxId(atx.ShortString()))

	//todo fetch from neighbour
	if atx.Nipst == nil {
		t.Panic("nil nipst in gossip")
		return
	}

	if err := t.syncer.FetchPoetProof(atx.GetPoetProofRef()); err != nil {
		t.Warning("received ATX (%v) with syntactically invalid or missing PoET proof (%x): %v",
			atx.ShortString(), atx.GetShortPoetProofRef(), err)
		return
	}

	id := atx.Id()
	events.Publish(events.NewAtx{Id: id.Hash32().String()})

	err = t.atxValidator.SyntacticallyValidateAtx(atx)
	events.Publish(events.ValidAtx{Id: atx.ShortString(), Valid: err == nil})
	if err != nil {
		t.Warning("received syntactically invalid ATX %v: %v", atx.ShortString(), err)
		// TODO: blacklist peer
		return
	}

	t.AtxPool.Put(atx.Id(), atx)
	data.ReportValidation(activation.AtxProtocol)
	t.With().Info("stored and propagated new syntactically valid ATX", log.AtxId(atx.ShortString()))
}

func (t *BlockBuilder) acceptBlockData() {
	for {
		select {

		case <-t.stopChan:
			return

		case id := <-t.beginRoundEvent:
			atxID, proofs, err := t.blockOracle.BlockEligible(id)
			if err != nil {
				t.With().Error("failed to check for block eligibility", log.LayerId(uint64(id)), log.Err(err))
				continue
			}
			if len(proofs) == 0 {
				t.With().Info("Notice: not eligible for blocks in layer", log.LayerId(uint64(id)))
				continue
			}
			// TODO: include multiple proofs in each block and weigh blocks where applicable

			txList := t.TransactionPool.PopItems(MaxTransactionsPerBlock)

			var atxList []types.AtxId
			for _, atx := range t.AtxPool.PopItems(MaxTransactionsPerBlock) {
				atxList = append(atxList, atx.Id())
			}

			for _, eligibilityProof := range proofs {
				blk, err := t.createBlock(types.LayerID(id), atxID, eligibilityProof, txList, atxList)
				if err != nil {
					t.Error("cannot create new block, %v ", err)
					continue
				}
				go func() {
					bytes, err := types.InterfaceToBytes(blk)
					if err != nil {
						t.Log.Error("cannot serialize block %v", err)
						return
					}
					err = t.network.Broadcast(config.NewBlockProtocol, bytes)
					if err != nil {
						t.Log.Error("cannot send block %v", err)
					}
				}()
			}
		}
	}
}
