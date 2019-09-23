package hare

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/spacemeshos/go-spacemesh/common/types"
	"hash/fnv"
	"sort"
	"strconv"
)

type bytes32 [32]byte

type blockID struct {
	types.BlockID
}
type instanceId types.LayerID

type messageType byte

// declare all known message types
const (
	status   messageType = 0
	proposal messageType = 1
	commit   messageType = 2
	notify   messageType = 3
	pre      messageType = 10
)

// declare round identifiers
const (
	preRound      = -1
	statusRound   = 0
	proposalRound = 1
	commitRound   = 2
	notifyRound   = 3
)

const defaultSetSize = 200

func (mType messageType) String() string {
	switch mType {
	case status:
		return "Status"
	case proposal:
		return "Proposal"
	case commit:
		return "Commit"
	case notify:
		return "Notify"
	case pre:
		return "PreRound"
	default:
		return "Unknown message type"
	}
}

func (id instanceId) Bytes() []byte {
	idInBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(idInBytes, uint32(id))

	return idInBytes
}

func newValue(value uint64) blockID {
	return blockID{types.BlockID(value)}
}

func (v blockID) Id() objectId {
	return objectId(v.BlockID)

}

func (v blockID) Bytes() []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(v.BlockID))
	return b
}

func (v blockID) String() string {
	return strconv.FormatUint(uint64(v.BlockID), 10)

}

func (b32 bytes32) Id() objectId {
	h := fnv.New32()
	h.Write(b32[:])
	return objectId(h.Sum32())
}

func (b32 bytes32) Bytes() []byte {
	return b32[:]
}

func (b32 bytes32) String() string {
	// TODO: should improve
	return string(b32.Id())
}

// Represents a unique set of Values
type Set struct {
	values    map[objectId]blockID
	id        objectId
	isIdValid bool
}

// Constructs an empty set
func NewSmallEmptySet() *Set {
	return NewEmptySet(defaultSetSize)
}

// Constructs an empty set
func NewEmptySet(expectedSize int) *Set {
	s := &Set{}
	s.values = make(map[objectId]blockID, expectedSize)
	s.id = 0
	s.isIdValid = false

	return s
}

// Constructs an empty set
func NewSetFromValues(values ...blockID) *Set {
	s := &Set{}
	s.values = make(map[objectId]blockID, len(values))
	for _, v := range values {
		s.Add(v)
	}
	s.id = 0
	s.isIdValid = false

	return s
}

// Constructs a new set from a 2D slice
// Each row represents a single value
func NewSet(data []uint64) *Set {
	s := &Set{}
	s.isIdValid = false

	s.values = make(map[objectId]blockID, len(data))
	for i := 0; i < len(data); i++ {
		bid := data[i]
		s.values[objectId(bid)] = blockID{types.BlockID(bid)}
	}

	return s
}

// Clones the set
func (s *Set) Clone() *Set {
	clone := NewEmptySet(len(s.values))
	for _, v := range s.values {
		clone.Add(v)
	}

	return clone
}

// Checks if a value is contained in the  set s
func (s *Set) Contains(id blockID) bool {
	_, exist := s.values[id.Id()]
	return exist
}

// Adds a value to the set if it doesn't exist already
func (s *Set) Add(id blockID) {
	if _, exist := s.values[id.Id()]; exist {
		return
	}

	s.isIdValid = false
	s.values[id.Id()] = id
}

// Removes a value from the set if exist
func (s *Set) Remove(id blockID) {
	if _, exist := s.values[id.Id()]; !exist {
		return
	}

	s.isIdValid = false
	delete(s.values, id.Id())
}

// Returns true if s and g represents the same set, false otherwise
func (s *Set) Equals(g *Set) bool {
	if len(s.values) != len(g.values) {
		return false
	}

	for _, bid := range s.values {
		if _, exist := g.values[bid.Id()]; !exist {
			return false
		}
	}

	return true
}

// ToSlice returns the array representation of the set
func (s *Set) ToSlice() []uint64 {
	// order keys
	keys := make([]objectId, len(s.values))
	i := 0
	for k := range s.values {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	l := make([]uint64, 0, len(s.values))
	for i := range keys {
		l = append(l, uint64(s.values[keys[i]].BlockID))
	}
	return l
}

func (s *Set) updateId() {
	// order keys
	keys := make([]objectId, len(s.values))
	i := 0
	for k := range s.values {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	// calc
	h := fnv.New32()
	for i := 0; i < len(keys); i++ {
		h.Write(s.values[keys[i]].Bytes())
	}

	// update
	s.id = objectId(h.Sum32())
	s.isIdValid = true
}

// Returns the objectId of the set
func (s *Set) Id() objectId {
	if !s.isIdValid {
		s.updateId()
	}

	return s.id
}

func (s *Set) String() string {
	// TODO: should improve
	b := new(bytes.Buffer)
	for _, v := range s.values {
		fmt.Fprintf(b, "%v,", v.Id())
	}
	if b.Len() >= 1 {
		return b.String()[:b.Len()-1]
	}
	return b.String()
}

// Check if s is a subset of g
func (s *Set) IsSubSetOf(g *Set) bool {
	for _, v := range s.values {
		if !g.Contains(v) {
			return false
		}
	}

	return true
}

// Returns the intersection set of s and g
func (s *Set) Intersection(g *Set) *Set {
	both := NewEmptySet(len(s.values))
	for _, v := range s.values {
		if g.Contains(v) {
			both.Add(v)
		}
	}

	return both
}

// Returns the union set of s and g
func (s *Set) Union(g *Set) *Set {
	union := NewEmptySet(len(s.values) + len(g.values))

	for _, v := range s.values {
		union.Add(v)
	}

	for _, v := range g.values {
		union.Add(v)
	}

	return union
}

// Returns the complement of s relatively to the world u
func (s *Set) Complement(u *Set) *Set {
	comp := NewEmptySet(len(u.values))
	for _, v := range u.values {
		if !s.Contains(v) {
			comp.Add(v)
		}
	}

	return comp
}

// Subtract g from s
func (s *Set) Subtract(g *Set) {
	for _, v := range g.values {
		s.Remove(v)
	}
}

// Returns the size of the set
func (s *Set) Size() int {
	return len(s.values)
}
