package hare

import (
	"errors"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/signing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func defaultValidator() *syntaxContextValidator {
	return newSyntaxContextValidator(signing.NewEdSigner(), lowThresh10, func(m *Msg) bool {
		return true
	}, &MockStateQuerier{true, nil}, 10, log.NewDefault("Validator"))
}

func TestMessageValidator_CommitStatus(t *testing.T) {
	assert.True(t, validateCommitType(BuildCommitMsg(generateSigning(t), NewEmptySet(lowDefaultSize))))
	assert.True(t, validateStatusType(BuildStatusMsg(generateSigning(t), NewEmptySet(lowDefaultSize))))
}

func TestMessageValidator_ValidateCertificate(t *testing.T) {
	validator := defaultValidator()
	assert.False(t, validator.validateCertificate(nil))
	cert := &certificate{}
	assert.False(t, validator.validateCertificate(cert))
	cert.AggMsgs = &aggregatedMessages{}
	assert.False(t, validator.validateCertificate(cert))
	msgs := make([]*Message, 0, validator.threshold)
	cert.AggMsgs.Messages = msgs
	assert.False(t, validator.validateCertificate(cert))
	msgs = append(msgs, &Message{})
	cert.AggMsgs.Messages = msgs
	assert.False(t, validator.validateCertificate(cert))
	cert.Values = NewSetFromValues(value1).ToSlice()
	assert.False(t, validator.validateCertificate(cert))

	msgs = make([]*Message, validator.threshold)
	for i := 0; i < validator.threshold; i++ {
		msgs[i] = BuildCommitMsg(generateSigning(t), NewDefaultEmptySet()).Message
	}
	cert.AggMsgs.Messages = msgs
	assert.True(t, validator.validateCertificate(cert))
}

func TestEligibilityValidator_validateRole(t *testing.T) {
	oracle := &mockRolacle{}
	ev := NewEligibilityValidator(oracle, 10, &mockIdProvider{}, 1, 5, log.NewDefault(""))
	ev.oracle = oracle
	res, err := ev.validateRole(nil)
	assert.NotNil(t, err)
	assert.False(t, res)
	m := BuildPreRoundMsg(generateSigning(t), NewDefaultEmptySet())
	m.InnerMsg = nil
	res, err = ev.validateRole(m)
	assert.NotNil(t, err)
	assert.False(t, res)
	m = BuildPreRoundMsg(generateSigning(t), NewDefaultEmptySet())
	oracle.isEligible = false
	res, err = ev.validateRole(m)
	assert.Nil(t, err)
	// TODO: remove comment after inceptions problem is addressed
	//assert.False(t, res)

	m.InnerMsg.InstanceId = 111
	myErr := errors.New("my error")
	ev.identityProvider = &mockIdProvider{myErr}
	res, err = ev.validateRole(m)
	assert.NotNil(t, err)
	assert.Equal(t, myErr, err)
	assert.False(t, res)

	oracle.err = myErr
	res, err = ev.validateRole(m)
	assert.NotNil(t, err)
	assert.Equal(t, myErr, err)
	assert.False(t, res)

	ev.identityProvider = &mockIdProvider{nil}
	oracle.err = nil
	res, err = ev.validateRole(m)
	assert.Nil(t, err)
	assert.False(t, res)

	oracle.isEligible = true
	m.InnerMsg.InstanceId = 111
	res, err = ev.validateRole(m)
	assert.Nil(t, err)
	assert.True(t, res)
}

func TestMessageValidator_IsStructureValid(t *testing.T) {
	validator := defaultValidator()
	assert.False(t, validator.SyntacticallyValidateMessage(nil))
	m := &Msg{&Message{}, nil}
	assert.False(t, validator.SyntacticallyValidateMessage(m))
	m.PubKey = generateSigning(t).PublicKey()
	assert.False(t, validator.SyntacticallyValidateMessage(m))
	m.InnerMsg = &innerMessage{}
	assert.False(t, validator.SyntacticallyValidateMessage(m))
	m.InnerMsg.Values = nil
	assert.False(t, validator.SyntacticallyValidateMessage(m))
	m.InnerMsg.Values = NewDefaultEmptySet().ToSlice()
	assert.False(t, validator.SyntacticallyValidateMessage(m))
}

func TestMessageValidator_Aggregated(t *testing.T) {
	validator := defaultValidator()
	assert.False(t, validator.validateAggregatedMessage(nil, nil))
	funcs := make([]func(m *Msg) bool, 0)
	assert.False(t, validator.validateAggregatedMessage(nil, funcs))

	agg := &aggregatedMessages{}
	assert.False(t, validator.validateAggregatedMessage(agg, funcs))
	msgs := make([]*Message, validator.threshold)
	for i := 0; i < validator.threshold; i++ {
		iMsg := BuildStatusMsg(generateSigning(t), NewSetFromValues(value1))
		msgs[i] = iMsg.Message
	}
	agg.Messages = msgs
	assert.True(t, validator.validateAggregatedMessage(agg, funcs))
	msgs[0].Sig = []byte{1}
	assert.False(t, validator.validateAggregatedMessage(agg, funcs))

	funcs = make([]func(m *Msg) bool, 1)
	funcs[0] = func(m *Msg) bool { return false }
	assert.False(t, validator.validateAggregatedMessage(agg, funcs))
}

func TestSyntaxContextValidator_PreRoundContext(t *testing.T) {
	r := require.New(t)
	validator := defaultValidator()
	ed := signing.NewEdSigner()
	pre := BuildPreRoundMsg(ed, NewDefaultEmptySet())
	for i := int32(0); i < 10; i++ {
		k := i * 4
		pre.InnerMsg.K = k
		e := validator.ContextuallyValidateMessage(pre, k)
		r.Nil(e)
	}
}

func TestSyntaxContextValidator_ContextuallyValidateMessageForIteration(t *testing.T) {
	r := require.New(t)
	v := defaultValidator()
	ed := signing.NewEdSigner()
	set := NewDefaultEmptySet()
	pre := BuildPreRoundMsg(ed, set)
	pre.InnerMsg.K = -1
	r.Nil(v.ContextuallyValidateMessage(pre, 1))

	status := BuildStatusMsg(ed, set)
	status.InnerMsg.K = 100
	r.Equal(errInvalidIter, v.ContextuallyValidateMessage(status, 1))

	proposal := BuildCommitMsg(ed, set)
	proposal.InnerMsg.K = 100
	r.Equal(errInvalidIter, v.ContextuallyValidateMessage(proposal, 1))

	commit := BuildCommitMsg(ed, set)
	commit.InnerMsg.K = 100
	r.Equal(errInvalidIter, v.ContextuallyValidateMessage(commit, 1))

	notify := BuildNotifyMsg(ed, set)
	notify.InnerMsg.K = 100
	r.Equal(errInvalidIter, v.ContextuallyValidateMessage(notify, 1))
}

func TestMessageValidator_ValidateMessage(t *testing.T) {
	proc := generateConsensusProcess(t)
	proc.advanceToNextRound()
	v := proc.validator
	b, err := proc.initDefaultBuilder(proc.s)
	assert.Nil(t, err)
	preround := b.SetType(pre).Sign(proc.signing).Build()
	preround.PubKey = proc.signing.PublicKey()
	assert.True(t, v.SyntacticallyValidateMessage(preround))
	e := v.ContextuallyValidateMessage(preround, 0)
	assert.Nil(t, e)
	b, err = proc.initDefaultBuilder(proc.s)
	assert.Nil(t, err)
	status := b.SetType(status).Sign(proc.signing).Build()
	status.PubKey = proc.signing.PublicKey()
	e = v.ContextuallyValidateMessage(status, 0)
	assert.Nil(t, e)
	assert.True(t, v.SyntacticallyValidateMessage(status))
}

func assertNoErr(r *require.Assertions, expect bool, actual bool, err error) {
	r.NoError(err)
	r.Equal(expect, actual)
}

func TestMessageValidator_SyntacticallyValidateMessage(t *testing.T) {
	validator := newSyntaxContextValidator(signing.NewEdSigner(), 1, validate, &MockStateQuerier{true, nil}, 10, log.NewDefault("Validator"))
	m := BuildPreRoundMsg(generateSigning(t), NewDefaultEmptySet())
	assert.False(t, validator.SyntacticallyValidateMessage(m))
	m = BuildPreRoundMsg(generateSigning(t), NewSetFromValues(value1))
	assert.True(t, validator.SyntacticallyValidateMessage(m))
}

func TestMessageValidator_validateSVPTypeA(t *testing.T) {
	m := buildProposalMsg(signing.NewEdSigner(), NewSetFromValues(value1, value2, value3), []byte{})
	s1 := NewSetFromValues(value1)
	s2 := NewSetFromValues(value3)
	s3 := NewSetFromValues(value1, value5)
	s4 := NewSetFromValues(value1, value4)
	v := defaultValidator()
	m.InnerMsg.Svp = buildSVP(-1, s1, s2, s3, s4)
	assert.False(t, v.validateSVPTypeA(m))
	s3 = NewSetFromValues(value2)
	m.InnerMsg.Svp = buildSVP(-1, s1, s2, s3)
	assert.True(t, v.validateSVPTypeA(m))
}

func TestMessageValidator_validateSVPTypeB(t *testing.T) {
	m := buildProposalMsg(signing.NewEdSigner(), NewSetFromValues(value1, value2, value3), []byte{})
	s1 := NewSetFromValues(value1)
	m.InnerMsg.Svp = buildSVP(-1, s1)
	s := NewSetFromValues(value1)
	m.InnerMsg.Values = s.ToSlice()
	v := defaultValidator()
	assert.False(t, v.validateSVPTypeB(m, NewSetFromValues(value5)))
	assert.True(t, v.validateSVPTypeB(m, NewSetFromValues(value1)))
}

func TestMessageValidator_validateSVP(t *testing.T) {
	validator := newSyntaxContextValidator(signing.NewEdSigner(), 1, validate, &MockStateQuerier{true, nil}, 10, log.NewDefault("Validator"))
	m := buildProposalMsg(signing.NewEdSigner(), NewSetFromValues(value1, value2, value3), []byte{})
	s1 := NewSetFromValues(value1)
	m.InnerMsg.Svp = buildSVP(-1, s1)
	m.InnerMsg.Svp.Messages[0].InnerMsg.Type = commit
	assert.False(t, validator.validateSVP(m))
	m.InnerMsg.Svp = buildSVP(-1, s1)
	m.InnerMsg.Svp.Messages[0].InnerMsg.K = 4
	assert.False(t, validator.validateSVP(m))
	m.InnerMsg.Svp = buildSVP(-1, s1)
	assert.False(t, validator.validateSVP(m))
	s2 := NewSetFromValues(value1, value2, value3)
	m.InnerMsg.Svp = buildSVP(-1, s2)
	assert.True(t, validator.validateSVP(m))
	m.InnerMsg.Svp = buildSVP(0, s1)
	assert.False(t, validator.validateSVP(m))
	m.InnerMsg.Svp = buildSVP(0, s2)
	assert.True(t, validator.validateSVP(m))
}

func buildSVP(ki int32, S ...*Set) *aggregatedMessages {
	msgs := make([]*Message, 0, len(S))
	for _, s := range S {
		msgs = append(msgs, buildStatusMsg(signing.NewEdSigner(), s, ki).Message)
	}

	svp := &aggregatedMessages{}
	svp.Messages = msgs
	return svp
}

func validateMatrix(t *testing.T, mType messageType, msgK int32, exp []error) {
	r := require.New(t)
	rounds := []int32{-1, 0, 1, 2, 3, 4, 5, 6, 7}
	v := defaultValidator()
	sgn := generateSigning(t)
	set := NewEmptySet(1)
	var m *Msg = nil
	switch mType {
	case status:
		m = BuildStatusMsg(sgn, set)
	case proposal:
		m = BuildProposalMsg(sgn, set)
	case commit:
		m = BuildCommitMsg(sgn, set)
	case notify:
		m = BuildNotifyMsg(sgn, set)
	default:
		panic("unexpected msg type")
	}

	for i, round := range rounds {
		m.InnerMsg.K = msgK
		e := v.ContextuallyValidateMessage(m, round)
		r.Equal(exp[i], e, "msgK=%v\tround=%v\texp=%v\tactual=%v", msgK, round, exp[i], e)
	}
}

func TestSyntaxContextValidator_StatusContextMatrix(t *testing.T) {
	msg0 := []error{errEarlyMsg, nil, errInvalidRound, errInvalidRound, errInvalidRound, errInvalidIter, errInvalidIter, errInvalidIter, errInvalidIter}
	msg4 := []error{errInvalidIter, errInvalidIter, errInvalidIter, errInvalidIter, errEarlyMsg, nil, errInvalidRound, errInvalidRound, errInvalidRound}
	validateMatrix(t, status, 0, msg0)
	validateMatrix(t, status, 4, msg4)
}

func TestSyntaxContextValidator_ProposalContextMatrix(t *testing.T) {
	msg1 := []error{errInvalidRound, errEarlyMsg, nil, nil, errInvalidRound, errInvalidIter, errInvalidIter, errInvalidIter, errInvalidIter}
	msg5 := []error{errInvalidIter, errInvalidIter, errInvalidIter, errInvalidIter, errInvalidIter, errEarlyMsg, nil, nil, errInvalidRound}
	validateMatrix(t, proposal, 1, msg1)
	validateMatrix(t, proposal, 5, msg5)
}

func TestSyntaxContextValidator_CommitContextMatrix(t *testing.T) {
	msg2 := []error{errInvalidRound, errInvalidRound, errEarlyMsg, nil, errInvalidRound, errInvalidIter, errInvalidIter, errInvalidIter, errInvalidIter}
	msg6 := []error{errInvalidIter, errInvalidIter, errInvalidIter, errInvalidIter, errInvalidIter, errInvalidRound, errEarlyMsg, nil, errInvalidRound}
	validateMatrix(t, commit, 2, msg2)
	validateMatrix(t, commit, 6, msg6)
}

func TestSyntaxContextValidator_NotifyContextMatrix(t *testing.T) {
	msg3 := []error{errInvalidRound, errInvalidRound, errInvalidRound, errEarlyMsg, nil, nil, nil, nil, nil}
	msg7 := []error{errInvalidIter, errInvalidIter, errInvalidIter, errInvalidIter, errInvalidIter, errInvalidRound, errInvalidRound, errEarlyMsg, nil}
	validateMatrix(t, notify, 3, msg3)
	validateMatrix(t, notify, 7, msg7)
}
