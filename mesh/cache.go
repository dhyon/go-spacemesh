package mesh

import (
	lru "github.com/hashicorp/golang-lru"

	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/log"
)

type blockCache struct {
	cap int
	*lru.Cache
}

func newBlockCache(cap int) blockCache {
	cache, err := lru.New(cap)
	if err != nil {
		log.Panic("could not initialize cache ", err)
	}
	return blockCache{Cache: cache, cap: cap}
}

func (bc blockCache) Cap() int {
	return bc.cap
}

func (bc blockCache) put(b *types.Block) {
	bc.Cache.Add(b.ID(), *b)
}

func (bc blockCache) Get(id types.BlockID) *types.Block {
	item, found := bc.Cache.Get(id)
	if !found {
		return nil
	}
	blk := item.(types.Block)
	return &blk
}
