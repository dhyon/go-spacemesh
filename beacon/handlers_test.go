package beacon

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/go-spacemesh/beacon/mocks"
	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/database"
	"github.com/spacemeshos/go-spacemesh/log/logtest"
	"github.com/spacemeshos/go-spacemesh/signing"
	"github.com/spacemeshos/go-spacemesh/timesync"
)

const peerID = "peer1"

var errUnknown = errors.New("unknown")

func clockNeverNotify(t *testing.T) layerClock {
	return timesync.NewClock(timesync.RealClock{}, time.Hour, time.Now(), logtest.New(t).WithName("clock"))
}

func createProtocolDriver(t *testing.T, epoch types.EpochID) (*ProtocolDriver, *mocks.MockactivationDB) {
	types.SetLayersPerEpoch(3)
	ctrl := gomock.NewController(t)
	mockDB := mocks.NewMockactivationDB(ctrl)
	edSgn := signing.NewEdSigner()
	cfg := UnitTestConfig()
	pd := &ProtocolDriver{
		logger:          logtest.New(t).WithName("Beacon"),
		clock:           clockNeverNotify(t),
		config:          cfg,
		atxDB:           mockDB,
		edSigner:        edSgn,
		edVerifier:      signing.NewEDVerifier(),
		vrfVerifier:     signing.VRFVerifier{},
		current:         newState(cfg),
		next:            newState(cfg),
		epochInProgress: epoch,
		running:         1,
		inProtocol:      1,
	}
	return pd, mockDB
}

func createProtocolDriverWithFirstRoundVotes(t *testing.T, epoch types.EpochID, signer signing.Signer) (*ProtocolDriver, *mocks.MockactivationDB, []types.Hash32) {
	pd, mockDB := createProtocolDriver(t, epoch)
	hash1 := types.HexToHash32("0x12345678")
	hash2 := types.HexToHash32("0x23456789")
	hash3 := types.HexToHash32("0x34567890")
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.current.setMinerFirstRoundVote(signer.PublicKey(), [][]byte{hash1.Bytes(), hash2.Bytes(), hash3.Bytes()})
	return pd, mockDB, []types.Hash32{hash1, hash2, hash3}
}

func setMockAtxDbExpectations(mockDB *mocks.MockactivationDB) {
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)
	mockDB.EXPECT().GetAtxHeader(gomock.Any()).Return(&types.ActivationTxHeader{
		NIPostChallenge: types.NIPostChallenge{
			StartTick: 1,
			EndTick:   3,
		},
		NumUnits: 5,
	}, nil).Times(1)
	mockDB.EXPECT().GetAtxTimestamp(gomock.Any()).Return(time.Now().Add(-1*time.Second), nil).Times(1)
}

func createProposal(t *testing.T, signer, vrfSigner signing.Signer, epoch types.EpochID, corruptSignature bool) *ProposalMessage {
	nodeID := types.NodeID{
		Key:          signer.PublicKey().String(),
		VRFPublicKey: vrfSigner.PublicKey().Bytes(),
	}
	sig := buildSignedProposal(context.TODO(), vrfSigner, epoch, logtest.New(t))
	msg := &ProposalMessage{
		NodeID:       nodeID,
		EpochID:      epoch,
		VRFSignature: sig,
	}
	if corruptSignature {
		msg.VRFSignature = sig[1:]
	}
	return msg
}

func checkProposalQueueSize(t *testing.T, pd *ProtocolDriver, numProposals int) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	assert.Equal(t, numProposals, len(pd.current.proposalChan))
}

func checkFutureProposalQueueSize(t *testing.T, pd *ProtocolDriver, numProposals int) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	assert.Equal(t, numProposals, len(pd.next.proposalChan))
}

func checkProposed(t *testing.T, pd *ProtocolDriver, signer signing.Signer, proposed bool) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	_, ok := pd.current.hasProposed[string(signer.PublicKey().Bytes())]
	assert.Equal(t, proposed, ok)
}

func checkProposals(t *testing.T, pd *ProtocolDriver, expected proposals) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	assert.EqualValues(t, expected, pd.current.incomingProposals)
}

func createFirstVote(t *testing.T, signer signing.Signer, epoch types.EpochID, valid, pValid [][]byte, corruptSignature bool) *FirstVotingMessage {
	msg := &FirstVotingMessage{
		FirstVotingMessageBody: FirstVotingMessageBody{
			EpochID:                   epoch,
			ValidProposals:            valid,
			PotentiallyValidProposals: pValid,
		},
	}
	sig := signMessage(signer, msg.FirstVotingMessageBody, logtest.New(t))
	if corruptSignature {
		msg.Signature = sig[1:]
	} else {
		msg.Signature = sig
	}
	return msg
}

func checkVoted(t *testing.T, pd *ProtocolDriver, signer signing.Signer, round types.RoundID, voted bool) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	_, ok := pd.current.hasVoted[round][string(signer.PublicKey().Bytes())]
	assert.Equal(t, voted, ok)
}

func checkFirstIncomingVotes(t *testing.T, pd *ProtocolDriver, expected map[string]proposalList) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	assert.EqualValues(t, expected, pd.current.firstRoundIncomingVotes)
}

func createFollowingVote(t *testing.T, signer signing.Signer, epoch types.EpochID, round types.RoundID, bitVector []byte, corruptSignature bool) *FollowingVotingMessage {
	msg := &FollowingVotingMessage{
		FollowingVotingMessageBody: FollowingVotingMessageBody{
			EpochID:        epoch,
			RoundID:        round,
			VotesBitVector: bitVector,
		},
	}
	sig := signMessage(signer, msg.FollowingVotingMessageBody, logtest.New(t))
	if corruptSignature {
		msg.Signature = sig[1:]
	} else {
		msg.Signature = sig
	}
	return msg
}

func checkVoteMargins(t *testing.T, pd *ProtocolDriver, expected map[string]*big.Int) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	assert.EqualValues(t, expected, pd.current.votesMargin)
}

func Test_HandleSerializedProposalMessage_Success(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)

	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)

	msg := createProposal(t, signer, vrfSigner, epoch, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedProposalMessage(context.TODO(), peerID, msgBytes)
	checkProposalQueueSize(t, pd, 1)
}

func Test_HandleSerializedProposalMessage_Shutdown(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)

	atomic.StoreUint64(&pd.running, 0)
	msg := createProposal(t, signer, vrfSigner, epoch, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedProposalMessage(context.TODO(), peerID, msgBytes)
	checkProposalQueueSize(t, pd, 0)
}

func Test_HandleSerializedProposalMessage_NotInProtocol(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)

	atomic.StoreUint64(&pd.inProtocol, 0)
	msg := createProposal(t, signer, vrfSigner, epoch, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedProposalMessage(context.TODO(), peerID, msgBytes)
	checkProposalQueueSize(t, pd, 0)
}

func Test_HandleSerializedProposalMessage_Corrupted(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)

	msg := createProposal(t, signer, vrfSigner, epoch, true)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedProposalMessage(context.TODO(), peerID, msgBytes[1:])
	checkProposalQueueSize(t, pd, 0)
}

func Test_HandleSerializedProposalMessage_EpochTooOld(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)

	msg := createProposal(t, signer, vrfSigner, epoch-1, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedProposalMessage(context.TODO(), peerID, msgBytes)
	checkProposalQueueSize(t, pd, 0)
	checkFutureProposalQueueSize(t, pd, 0)
}

func Test_HandleSerializedProposalMessage_NextEpoch(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)

	msg := createProposal(t, signer, vrfSigner, epoch+1, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedProposalMessage(context.TODO(), peerID, msgBytes)
	checkProposalQueueSize(t, pd, 0)
	checkFutureProposalQueueSize(t, pd, 1)
}

func Test_HandleSerializedProposalMessage_NextEpochFull(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)

	for i := 0; i < proposalChanCapacity; i++ {
		msg := createProposal(t, signer, vrfSigner, epoch+1, false)
		msgBytes, err := types.InterfaceToBytes(msg)
		require.NoError(t, err)

		pd.HandleSerializedProposalMessage(context.TODO(), peerID, msgBytes)
	}
	checkProposalQueueSize(t, pd, 0)
	checkFutureProposalQueueSize(t, pd, proposalChanCapacity)

	// now try to overflow channel for the next epoch
	msg := createProposal(t, signer, vrfSigner, epoch+1, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedProposalMessage(context.TODO(), peerID, msgBytes)
	checkProposalQueueSize(t, pd, 0)
	checkFutureProposalQueueSize(t, pd, proposalChanCapacity)
}

func Test_HandleSerializedProposalMessage_EpochTooFarAhead(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)

	msg := createProposal(t, signer, vrfSigner, epoch+2, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedProposalMessage(context.TODO(), peerID, msgBytes)
	checkProposalQueueSize(t, pd, 0)
	checkFutureProposalQueueSize(t, pd, 0)
}

func mockProposalChecker(t *testing.T, pd *ProtocolDriver, success bool, calls int) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockChecker := mocks.NewMockeligibilityChecker(ctrl)
	mockChecker.EXPECT().IsProposalEligible(gomock.Any()).Return(success).Times(calls)
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.current.proposalChecker = mockChecker
}

func Test_handleProposalMessage_Success(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, mockDB := createProtocolDriver(t, epoch)
	setMockAtxDbExpectations(mockDB)
	signer1 := signing.NewEdSigner()
	vrfSigner1, _, err := signing.NewVRFSigner(signer1.Sign(signer1.PublicKey().Bytes()))
	require.NoError(t, err)
	msg1 := createProposal(t, signer1, vrfSigner1, epoch, false)
	mockProposalChecker(t, pd, true, 1)
	err = pd.handleProposalMessage(context.TODO(), *msg1, time.Now())
	assert.NoError(t, err)
	checkProposed(t, pd, vrfSigner1, true)

	// make the next proposal late
	pd.markProposalPhaseFinished(epoch)

	setMockAtxDbExpectations(mockDB)
	signer2 := signing.NewEdSigner()
	vrfSigner2, _, err := signing.NewVRFSigner(signer2.Sign(signer2.PublicKey().Bytes()))
	require.NoError(t, err)
	msg2 := createProposal(t, signer2, vrfSigner2, epoch, false)
	mockProposalChecker(t, pd, true, 1)
	err = pd.handleProposalMessage(context.TODO(), *msg2, time.Now())
	assert.NoError(t, err)
	checkProposed(t, pd, vrfSigner2, true)

	p1 := msg1.VRFSignature[:types.BeaconSize]
	p2 := msg2.VRFSignature[:types.BeaconSize]
	expectedProposals := proposals{
		valid:            proposalSet{string(p1): struct{}{}},
		potentiallyValid: proposalSet{string(p2): struct{}{}},
	}
	checkProposals(t, pd, expectedProposals)
}

func Test_handleProposalMessage_BadSignature(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)

	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)
	msg := createProposal(t, signer, vrfSigner, epoch, true)

	err = pd.handleProposalMessage(context.TODO(), *msg, time.Now())
	assert.ErrorIs(t, err, errVRFNotVerified)

	checkProposed(t, pd, vrfSigner, false)
	checkProposals(t, pd, proposals{})
}

func Test_handleProposalMessage_AlreadyProposed(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, mockDB := createProtocolDriver(t, epoch)
	setMockAtxDbExpectations(mockDB)
	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)
	msg := createProposal(t, signer, vrfSigner, epoch, false)

	mockProposalChecker(t, pd, true, 1)

	err = pd.handleProposalMessage(context.TODO(), *msg, time.Now())
	assert.NoError(t, err)

	checkProposed(t, pd, vrfSigner, true)
	p := msg.VRFSignature[:types.BeaconSize]
	expectedProposals := proposals{
		valid: proposalSet{string(p): struct{}{}},
	}
	checkProposals(t, pd, expectedProposals)

	// now make the same proposal again
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)

	err = pd.handleProposalMessage(context.TODO(), *msg, time.Now())
	assert.ErrorIs(t, err, errAlreadyProposed)
	checkProposed(t, pd, vrfSigner, true)
	checkProposals(t, pd, expectedProposals)
}

func Test_handleProposalMessage_ProposalNotEligible(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)

	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)
	msg := createProposal(t, signer, vrfSigner, epoch, false)

	mockProposalChecker(t, pd, false, 1)

	err = pd.handleProposalMessage(context.TODO(), *msg, time.Now())
	assert.ErrorIs(t, err, errProposalDoesntPassThreshold)

	checkProposed(t, pd, vrfSigner, true)
	checkProposals(t, pd, proposals{})
}

func Test_handleProposalMessage_MinerMissingATX(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID{}, database.ErrNotFound).Times(1)

	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)
	msg := createProposal(t, signer, vrfSigner, epoch, false)

	err = pd.handleProposalMessage(context.TODO(), *msg, time.Now())
	assert.ErrorIs(t, err, errMinerATXNotFound)

	checkProposed(t, pd, vrfSigner, false)
	checkProposals(t, pd, proposals{})
}

func Test_handleProposalMessage_ATXLookupError(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID{}, errUnknown).Times(1)

	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)
	msg := createProposal(t, signer, vrfSigner, epoch, false)

	err = pd.handleProposalMessage(context.TODO(), *msg, time.Now())
	assert.ErrorIs(t, err, errUnknown)

	checkProposed(t, pd, vrfSigner, false)
	checkProposals(t, pd, proposals{})
}

func Test_handleProposalMessage_ATXHeaderLookupError(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)
	mockDB.EXPECT().GetAtxHeader(gomock.Any()).Return(nil, errUnknown).Times(1)

	mockProposalChecker(t, pd, true, 1)

	signer := signing.NewEdSigner()
	vrfSigner, _, err := signing.NewVRFSigner(signer.Sign(signer.PublicKey().Bytes()))
	require.NoError(t, err)

	msg := createProposal(t, signer, vrfSigner, epoch, false)

	err = pd.handleProposalMessage(context.TODO(), *msg, time.Now())
	assert.ErrorIs(t, err, errUnknown)

	checkProposed(t, pd, vrfSigner, true)
	checkProposals(t, pd, proposals{})
}

func Test_HandleSerializedFirstVotingMessage_Success(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := [][]byte{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := [][]byte{types.HexToHash32("0x23456789").Bytes()}

	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)
	mockDB.EXPECT().GetAtxHeader(gomock.Any()).Return(&types.ActivationTxHeader{
		NIPostChallenge: types.NIPostChallenge{
			StartTick: 1,
			EndTick:   3,
		},
		NumUnits: 5,
	}, nil).Times(1)

	signer := signing.NewEdSigner()

	msg := createFirstVote(t, signer, epoch, valid, pValid, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedFirstVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, types.FirstRound, true)
	expected := map[string]proposalList{
		string(signer.PublicKey().Bytes()): append(valid, pValid...),
	}
	checkFirstIncomingVotes(t, pd, expected)
}

func Test_HandleSerializedFirstVotingMessage_Shutdown(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := [][]byte{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := [][]byte{types.HexToHash32("0x23456789").Bytes()}

	pd, _ := createProtocolDriver(t, epoch)
	atomic.StoreUint64(&pd.running, 0)

	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedFirstVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, types.FirstRound, false)
	checkFirstIncomingVotes(t, pd, map[string]proposalList{})
}

func Test_HandleSerializedFirstVotingMessage_NotInProtocol(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := proposalList{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := proposalList{types.HexToHash32("0x23456789").Bytes()}

	pd, _ := createProtocolDriver(t, epoch)
	atomic.StoreUint64(&pd.inProtocol, 0)

	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedFirstVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, types.FirstRound, false)
	checkFirstIncomingVotes(t, pd, map[string]proposalList{})
}

func Test_HandleSerializedFirstVotingMessage_TooLate(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := proposalList{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := proposalList{types.HexToHash32("0x23456789").Bytes()}

	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.setRoundInProgress(types.RoundID(1))
	pd.HandleSerializedFirstVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, types.FirstRound, false)
	checkFirstIncomingVotes(t, pd, map[string]proposalList{})
}

func Test_HandleSerializedFirstVotingMessage_CorruptedGossipMsg(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := proposalList{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := proposalList{types.HexToHash32("0x23456789").Bytes()}

	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, true)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedFirstVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, types.FirstRound, false)
	checkFirstIncomingVotes(t, pd, map[string]proposalList{})
}

func Test_HandleSerializedFirstVotingMessage_WrongEpoch(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := proposalList{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := proposalList{types.HexToHash32("0x23456789").Bytes()}

	pd, _ := createProtocolDriver(t, epoch+1)
	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.HandleSerializedFirstVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, types.FirstRound, false)
	checkFirstIncomingVotes(t, pd, map[string]proposalList{})
}

func Test_handleFirstVotingMessage_Success(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := proposalList{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := proposalList{types.HexToHash32("0x23456789").Bytes()}

	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)
	mockDB.EXPECT().GetAtxHeader(gomock.Any()).Return(&types.ActivationTxHeader{
		NIPostChallenge: types.NIPostChallenge{
			StartTick: 1,
			EndTick:   3,
		},
		NumUnits: 5,
	}, nil).Times(1)

	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, false)

	err := pd.handleFirstVotingMessage(context.TODO(), *msg)
	assert.NoError(t, err)

	checkVoted(t, pd, signer, types.FirstRound, true)
	expected := map[string]proposalList{
		string(signer.PublicKey().Bytes()): append(valid, pValid...),
	}
	checkFirstIncomingVotes(t, pd, expected)
}

func Test_handleFirstVotingMessage_FailedToExtractPK(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := proposalList{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := proposalList{types.HexToHash32("0x23456789").Bytes()}

	pd, _ := createProtocolDriver(t, epoch)
	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, true)

	err := pd.handleFirstVotingMessage(context.TODO(), *msg)
	assert.Contains(t, err.Error(), "bad signature format")

	checkVoted(t, pd, signer, types.FirstRound, false)
	checkFirstIncomingVotes(t, pd, map[string]proposalList{})
}

func Test_handleFirstVotingMessage_AlreadyVoted(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := proposalList{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := proposalList{types.HexToHash32("0x23456789").Bytes()}

	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)
	mockDB.EXPECT().GetAtxHeader(gomock.Any()).Return(&types.ActivationTxHeader{
		NIPostChallenge: types.NIPostChallenge{
			StartTick: 1,
			EndTick:   3,
		},
		NumUnits: 5,
	}, nil).Times(1)

	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, false)

	err := pd.handleFirstVotingMessage(context.TODO(), *msg)
	assert.NoError(t, err)

	checkVoted(t, pd, signer, types.FirstRound, true)
	expected := map[string]proposalList{
		string(signer.PublicKey().Bytes()): append(valid, pValid...),
	}
	checkFirstIncomingVotes(t, pd, expected)

	// now vote again
	err = pd.handleFirstVotingMessage(context.TODO(), *msg)
	assert.ErrorIs(t, err, errAlreadyVoted)

	checkVoted(t, pd, signer, types.FirstRound, true)
	checkFirstIncomingVotes(t, pd, expected)
}

func Test_handleFirstVotingMessage_MinerMissingATX(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := proposalList{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := proposalList{types.HexToHash32("0x23456789").Bytes()}

	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID{}, database.ErrNotFound).Times(1)

	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, false)

	err := pd.handleFirstVotingMessage(context.TODO(), *msg)
	assert.ErrorIs(t, err, errMinerATXNotFound)

	checkVoted(t, pd, signer, types.FirstRound, true)
	checkFirstIncomingVotes(t, pd, map[string]proposalList{})
}

func Test_handleFirstVotingMessage_ATXLookupError(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := proposalList{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := proposalList{types.HexToHash32("0x23456789").Bytes()}

	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID{}, errUnknown).Times(1)

	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, false)

	err := pd.handleFirstVotingMessage(context.TODO(), *msg)
	assert.ErrorIs(t, err, errUnknown)

	checkVoted(t, pd, signer, types.FirstRound, true)
	checkFirstIncomingVotes(t, pd, map[string]proposalList{})
}

func Test_handleFirstVotingMessage_ATXHeaderLookupError(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	valid := proposalList{types.HexToHash32("0x12345678").Bytes(), types.HexToHash32("0x87654321").Bytes()}
	pValid := proposalList{types.HexToHash32("0x23456789").Bytes()}

	pd, mockDB := createProtocolDriver(t, epoch)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)
	mockDB.EXPECT().GetAtxHeader(gomock.Any()).Return(nil, errUnknown).Times(1)

	signer := signing.NewEdSigner()
	msg := createFirstVote(t, signer, epoch, valid, pValid, false)

	err := pd.handleFirstVotingMessage(context.TODO(), *msg)
	assert.ErrorIs(t, err, errUnknown)

	checkVoted(t, pd, signer, types.FirstRound, true)
	checkFirstIncomingVotes(t, pd, map[string]proposalList{})
}

func Test_HandleSerializedFollowingVotingMessage_Success(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)

	signer := signing.NewEdSigner()
	pd, mockDB, hashes := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)
	mockDB.EXPECT().GetAtxHeader(gomock.Any()).Return(&types.ActivationTxHeader{
		NIPostChallenge: types.NIPostChallenge{
			StartTick: 1,
			EndTick:   3,
		},
		NumUnits: 5,
	}, nil).Times(1)

	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.setRoundInProgress(round)
	pd.HandleSerializedFollowingVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, round, true)
	expected := make(map[string]*big.Int, len(hashes))
	for i, hash := range hashes {
		if i == 0 || i == 2 {
			expected[string(hash.Bytes())] = big.NewInt(10)
		} else {
			expected[string(hash.Bytes())] = big.NewInt(-10)
		}
	}
	checkVoteMargins(t, pd, expected)
}

func Test_HandleSerializedFollowingVotingMessage_Shutdown(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)

	signer := signing.NewEdSigner()
	pd, _, _ := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	atomic.StoreUint64(&pd.running, 0)

	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.setRoundInProgress(round)
	pd.HandleSerializedFollowingVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, round, false)
	checkVoteMargins(t, pd, map[string]*big.Int{})
}

func Test_HandleSerializedFollowingVotingMessage_NotInProtocol(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)

	signer := signing.NewEdSigner()
	pd, _, _ := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	atomic.StoreUint64(&pd.inProtocol, 0)
	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.setRoundInProgress(round)
	pd.HandleSerializedFollowingVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, round, false)
	checkVoteMargins(t, pd, map[string]*big.Int{})
}

func Test_HandleSerializedFollowingVotingMessage_TooEarly(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)
	signer := signing.NewEdSigner()
	pd, _, _ := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.setRoundInProgress(round - 1)
	pd.HandleSerializedFollowingVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, round, false)
	checkVoteMargins(t, pd, map[string]*big.Int{})
}

func Test_HandleSerializedFollowingVotingMessage_CorruptedGossipMsg(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)
	signer := signing.NewEdSigner()
	pd, _, _ := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, true)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.setRoundInProgress(round)
	pd.HandleSerializedFollowingVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, round, false)
	checkVoteMargins(t, pd, map[string]*big.Int{})
}

func Test_HandleSerializedFollowingVotingMessage_WrongEpoch(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)
	signer := signing.NewEdSigner()
	pd, _, _ := createProtocolDriverWithFirstRoundVotes(t, epoch+1, signer)
	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, false)
	msgBytes, err := types.InterfaceToBytes(msg)
	require.NoError(t, err)

	pd.setRoundInProgress(round)
	pd.HandleSerializedFollowingVotingMessage(context.TODO(), peerID, msgBytes)
	checkVoted(t, pd, signer, round, false)
	checkVoteMargins(t, pd, map[string]*big.Int{})
}

func Test_handleFollowingVotingMessage_Success(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)
	signer := signing.NewEdSigner()
	pd, mockDB, hashes := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)
	mockDB.EXPECT().GetAtxHeader(gomock.Any()).Return(&types.ActivationTxHeader{
		NIPostChallenge: types.NIPostChallenge{
			StartTick: 1,
			EndTick:   3,
		},
		NumUnits: 5,
	}, nil).Times(1)

	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, false)

	pd.setRoundInProgress(round)
	err := pd.handleFollowingVotingMessage(context.TODO(), *msg)
	assert.NoError(t, err)

	checkVoted(t, pd, signer, round, true)
	expected := make(map[string]*big.Int, len(hashes))
	for i, hash := range hashes {
		if i == 0 || i == 2 {
			expected[string(hash.Bytes())] = big.NewInt(10)
		} else {
			expected[string(hash.Bytes())] = big.NewInt(-10)
		}
	}
	checkVoteMargins(t, pd, expected)
}

func Test_handleFollowingVotingMessage_FailedToExtractPK(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)
	signer := signing.NewEdSigner()
	pd, _, _ := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, true)

	pd.setRoundInProgress(round)
	err := pd.handleFollowingVotingMessage(context.TODO(), *msg)
	assert.Contains(t, err.Error(), "bad signature format")

	checkVoted(t, pd, signer, round, false)
	checkVoteMargins(t, pd, map[string]*big.Int{})
}

func Test_handleFollowingVotingMessage_AlreadyVoted(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)
	signer := signing.NewEdSigner()
	pd, mockDB, hashes := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)
	mockDB.EXPECT().GetAtxHeader(gomock.Any()).Return(&types.ActivationTxHeader{
		NIPostChallenge: types.NIPostChallenge{
			StartTick: 1,
			EndTick:   3,
		},
		NumUnits: 5,
	}, nil).Times(1)

	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, false)

	pd.setRoundInProgress(round)
	err := pd.handleFollowingVotingMessage(context.TODO(), *msg)
	assert.NoError(t, err)

	checkVoted(t, pd, signer, round, true)
	expected := make(map[string]*big.Int, len(hashes))
	for i, hash := range hashes {
		if i == 0 || i == 2 {
			expected[string(hash.Bytes())] = big.NewInt(10)
		} else {
			expected[string(hash.Bytes())] = big.NewInt(-10)
		}
	}
	checkVoteMargins(t, pd, expected)

	// now vote again
	err = pd.handleFollowingVotingMessage(context.TODO(), *msg)
	assert.ErrorIs(t, err, errAlreadyVoted)

	checkVoted(t, pd, signer, round, true)
	checkVoteMargins(t, pd, expected)
}

func Test_handleFollowingVotingMessage_MinerMissingATX(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)
	signer := signing.NewEdSigner()
	pd, mockDB, _ := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID{}, database.ErrNotFound).Times(1)
	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, false)

	pd.setRoundInProgress(round)
	err := pd.handleFollowingVotingMessage(context.TODO(), *msg)
	assert.ErrorIs(t, err, errMinerATXNotFound)

	checkVoted(t, pd, signer, round, true)
	checkVoteMargins(t, pd, map[string]*big.Int{})
}

func Test_handleFollowingVotingMessage_ATXLookupError(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)
	signer := signing.NewEdSigner()
	pd, mockDB, _ := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID{}, errUnknown).Times(1)
	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, false)

	pd.setRoundInProgress(round)
	err := pd.handleFollowingVotingMessage(context.TODO(), *msg)
	assert.ErrorIs(t, err, errUnknown)

	checkVoted(t, pd, signer, round, true)
	checkVoteMargins(t, pd, map[string]*big.Int{})
}

func Test_handleFollowingVotingMessage_ATXHeaderLookupError(t *testing.T) {
	t.Parallel()

	const epoch = types.EpochID(10)
	const round = types.RoundID(5)
	signer := signing.NewEdSigner()
	pd, mockDB, _ := createProtocolDriverWithFirstRoundVotes(t, epoch, signer)
	mockDB.EXPECT().GetNodeAtxIDForEpoch(gomock.Any(), gomock.Any()).
		Return(types.ATXID(types.HexToHash32("0x22345678")), nil).Times(1)
	mockDB.EXPECT().GetAtxHeader(gomock.Any()).Return(nil, errUnknown).Times(1)
	// this msg will contain a bit vector that set bit 0 and 2
	msg := createFollowingVote(t, signer, epoch, round, []byte{0b101}, false)

	pd.setRoundInProgress(round)
	err := pd.handleFollowingVotingMessage(context.TODO(), *msg)
	assert.ErrorIs(t, err, errUnknown)

	checkVoted(t, pd, signer, round, true)
	checkVoteMargins(t, pd, map[string]*big.Int{})
}

func Test_uniqueFollowingVotingMessages(t *testing.T) {
	round := types.RoundID(3)
	votesBitVector := []byte{0b101}
	edSgn := signing.NewEdSigner()
	msg1 := FollowingVotingMessage{
		FollowingVotingMessageBody: FollowingVotingMessageBody{
			RoundID:        round,
			VotesBitVector: votesBitVector,
		},
	}
	msg1.Signature = signMessage(edSgn, msg1.FollowingVotingMessageBody, logtest.New(t))
	data1, err := types.InterfaceToBytes(msg1)
	require.NoError(t, err)

	msg2 := FollowingVotingMessage{
		FollowingVotingMessageBody: FollowingVotingMessageBody{
			RoundID:        round,
			VotesBitVector: votesBitVector,
		},
	}
	msg2.Signature = signMessage(edSgn, msg2.FollowingVotingMessageBody, logtest.New(t))
	data2, err := types.InterfaceToBytes(msg2)
	require.NoError(t, err)

	// without EpochID, we cannot tell the following messages apart
	assert.Equal(t, data1, data2)

	msg1.EpochID = types.EpochID(5)
	msg1.Signature = signMessage(edSgn, msg1.FollowingVotingMessageBody, logtest.New(t))
	data1, err = types.InterfaceToBytes(msg1)
	require.NoError(t, err)

	msg2.EpochID = msg1.EpochID + 1
	msg2.Signature = signMessage(edSgn, msg2.FollowingVotingMessageBody, logtest.New(t))
	data2, err = types.InterfaceToBytes(msg2)
	require.NoError(t, err)

	// with EpochID, voting messages from the same miner with the same bit vector will
	// not be considered duplicate gossip messages.
	assert.NotEqual(t, data1, data2)
}
