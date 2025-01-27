package types

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/spacemeshos/ed25519"

	"github.com/spacemeshos/go-spacemesh/codec"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/signing"
)

const (
	// ProposalIDSize in bytes.
	// FIXME(dshulyak) why do we cast to hash32 when returning bytes?
	// probably required for fetching by hash between peers.
	ProposalIDSize = Hash32Length
)

// ProposalID is a 20-byte sha256 sum of the serialized ballot used to identify a Proposal.
type ProposalID Hash20

// EmptyProposalID is a canonical empty ProposalID.
var EmptyProposalID = ProposalID{}

// Proposal contains the smesher's signed content proposal for a given layer and vote on the mesh history.
// Proposal is ephemeral and will be discarded after the unified content block is created. the Ballot within
// the Proposal will remain in the mesh.
type Proposal struct {
	// the content proposal for a given layer and the votes on the mesh history
	InnerProposal
	// smesher's signature on InnerProposal
	Signature []byte

	// the following fields are kept private and from being serialized
	proposalID ProposalID
}

// InnerProposal contains a smesher's content proposal for layer and its votes on the mesh history.
// this structure is serialized and signed to produce the signature in Proposal.
type InnerProposal struct {
	// smesher's votes on the mesh history
	Ballot
	// smesher's content proposal for a layer
	TxIDs []TransactionID
}

// Initialize calculates and sets the Proposal's cached proposalID.
// this should be called once all the other fields of the Proposal are set.
func (p *Proposal) Initialize() error {
	if p.ID() != EmptyProposalID {
		return fmt.Errorf("proposal already initialized")
	}

	if err := p.Ballot.Initialize(); err != nil {
		return err
	}

	// check proposal signature consistent with ballot's
	pubkey, err := ed25519.ExtractPublicKey(p.Bytes(), p.Signature)
	if err != nil {
		return fmt.Errorf("proposal extract key: %w", err)
	}
	pPubKey := signing.NewPublicKey(pubkey)
	if !p.Ballot.SmesherID().Equals(pPubKey) {
		return fmt.Errorf("inconsistent smesher in proposal %v and ballot %v", pPubKey.ShortString(), p.Ballot.SmesherID().ShortString())
	}

	p.proposalID = ProposalID(CalcHash32(p.Bytes()).ToHash20())
	return nil
}

// Bytes returns the serialization of the InnerProposal.
func (p *Proposal) Bytes() []byte {
	bytes, err := codec.Encode(p.InnerProposal)
	if err != nil {
		log.Panic("failed to serialize proposal: %v", err)
	}
	return bytes
}

// ID returns the ProposalID.
func (p *Proposal) ID() ProposalID {
	return p.proposalID
}

// MarshalLogObject implements logging interface.
func (p *Proposal) MarshalLogObject(encoder log.ObjectEncoder) error {
	encoder.AddString("proposal_id", p.ID().String())
	p.Ballot.MarshalLogObject(encoder)
	return nil
}

// String returns a short prefix of the hex representation of the ID.
func (id ProposalID) String() string {
	return id.AsHash32().ShortString()
}

// Bytes returns the ProposalID as a byte slice.
func (id ProposalID) Bytes() []byte {
	return id.AsHash32().Bytes()
}

// AsHash32 returns a Hash32 whose first 20 bytes are the bytes of this ProposalID, it is right-padded with zeros.
func (id ProposalID) AsHash32() Hash32 {
	return Hash20(id).ToHash32()
}

// Field returns a log field. Implements the LoggableField interface.
func (id ProposalID) Field() log.Field {
	return log.String("proposal_id", id.String())
}

// Compare returns true if other (the given ProposalID) is less than this ProposalID, by lexicographic comparison.
func (id ProposalID) Compare(other ProposalID) bool {
	return bytes.Compare(id.Bytes(), other.Bytes()) < 0
}

// ToProposalIDs returns a slice of ProposalID corresponding to the given proposals.
func ToProposalIDs(proposals []*Proposal) []ProposalID {
	ids := make([]ProposalID, 0, len(proposals))
	for _, p := range proposals {
		ids = append(ids, p.ID())
	}
	return ids
}

// SortProposals sorts a list of Proposal in their ID's lexicographic order, in-place.
func SortProposals(proposals []*Proposal) []*Proposal {
	sort.Slice(proposals, func(i, j int) bool { return proposals[i].ID().Compare(proposals[j].ID()) })
	return proposals
}

// SortProposalIDs sorts a list of ProposalID in lexicographic order, in-place.
func SortProposalIDs(ids []ProposalID) []ProposalID {
	sort.Slice(ids, func(i, j int) bool { return ids[i].Compare(ids[j]) })
	return ids
}

// ProposalIDsToHashes turns a list of ProposalID into their Hash32 representation.
func ProposalIDsToHashes(ids []ProposalID) []Hash32 {
	hashes := make([]Hash32, 0, len(ids))
	for _, id := range ids {
		hashes = append(hashes, id.AsHash32())
	}
	return hashes
}

// DBProposal is a Proposal structure stored in DB to skip signature verification.
type DBProposal struct {
	// NOTE(dshulyak) this is a bit redundant to store ID here as well but less likely
	// to break if in future key for database will be changed
	ID         ProposalID
	BallotID   BallotID
	LayerIndex LayerID
	TxIDs      []TransactionID
	Signature  []byte
}

// ToProposal creates a Proposal from data that is stored locally.
func (b *DBProposal) ToProposal(ballot *Ballot) *Proposal {
	return &Proposal{
		InnerProposal: InnerProposal{
			Ballot: *ballot,
			TxIDs:  b.TxIDs,
		},
		Signature:  b.Signature,
		proposalID: b.ID,
	}
}
