package sync

import (
	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/p2p"
	"github.com/spacemeshos/go-spacemesh/p2p/p2pcrypto"
	"github.com/spacemeshos/go-spacemesh/p2p/server"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

type RequestFactory func(s WorkerInfra, peer p2p.Peer) (chan interface{}, error)
type FetchRequestFactory func(s WorkerInfra, peer p2p.Peer, id interface{}) (chan []Item, error)
type BlockRequestFactory func(s WorkerInfra, peer p2p.Peer, id interface{}) (chan interface{}, error)

type WorkerInfra interface {
	GetPeers() []p2p.Peer
	SendRequest(msgType server.MessageType, payload []byte, address p2pcrypto.PublicKey, resHandler func(msg []byte)) error
	GetTimeout() time.Duration
	log.Logger
}

type workerInfra struct {
	RequestTimeout time.Duration
	*server.MessageServer
	p2p.Peers
}

func (ms *workerInfra) Close() {
	ms.MessageServer.Close()
	ms.Peers.Close()
}

func (ms *workerInfra) GetTimeout() time.Duration {
	return ms.RequestTimeout
}

type worker struct {
	*sync.Once
	*sync.WaitGroup
	work      func()
	workCount *int32
	output    chan interface{}
	log.Logger
}

func (w *worker) Work() {
	w.Debug("worker work")
	w.work()
	atomic.AddInt32(w.workCount, -1)
	w.Debug("worker done")
	if atomic.LoadInt32(w.workCount) == 0 { //close once everyone is finished
		w.Debug("worker teardown")
		w.Do(func() { close(w.output) })
	}
}

func (w *worker) Clone() *worker {
	return &worker{Logger: w.Logger, Once: w.Once, workCount: w.workCount, output: w.output, work: w.work}
}

func NewPeersWorker(s WorkerInfra, peers []p2p.Peer, mu *sync.Once, reqFactory RequestFactory) (worker, chan interface{}) {
	count := int32(1)
	numOfpeers := len(peers)
	output := make(chan interface{}, numOfpeers)
	peerfuncs := []func(){}
	wg := sync.WaitGroup{}
	wg.Add(len(peers))
	lg := s.WithName("peersWrkr")
	for _, p := range peers {
		peer := p
		peerFunc := func() {
			defer wg.Done()
			lg.Info("send request Peer: %v", peer)
			ch, err := reqFactory(s, peer)
			if err != nil {
				s.Error("request failed, ", err)
				return
			}

			timeout := time.After(s.GetTimeout())
			select {
			case <-timeout:
				lg.Error("request to %v timed out", peer)
				return
			case v := <-ch:
				if v != nil {
					lg.Info("Peer: %v responded", peer)
					output <- v
				}
			}
		}

		peerfuncs = append(peerfuncs, peerFunc)
	}

	wrkFunc := func() {
		for _, p := range peerfuncs {
			go p()
		}
		wg.Wait()
	}

	worker := worker{Logger: lg, Once: mu, WaitGroup: &sync.WaitGroup{}, workCount: &count, output: output, work: wrkFunc}

	return worker, output

}

func NewNeighborhoodWorker(s WorkerInfra, count int, reqFactory RequestFactory) worker {
	output := make(chan interface{}, count)
	acount := int32(count)
	mu := &sync.Once{}
	lg := s.WithName("HoodWrker")
	workFunc := func() {
		peers := s.GetPeers()
		for _, p := range peers {
			peer := p
			lg.Info("send request Peer: %v", peer)
			ch, _ := reqFactory(s, peer)
			timeout := time.After(s.GetTimeout())
			select {
			case <-timeout:
				lg.Error("request to %v timed out", peer)
			case v := <-ch:
				if v != nil {
					lg.Info("Peer: %v responded ", peer)
					lg.Debug("Peer: %v response was  %v", v)
					output <- v
					return
				}
			}
		}
	}

	return worker{Logger: lg, Once: mu, WaitGroup: &sync.WaitGroup{}, workCount: &acount, output: output, work: workFunc}

}

func NewFetchWorker(s WorkerInfra, count int, reqFactory FetchRequestFactory, idsChan chan []types.Hash32) worker {
	output := make(chan interface{}, count)
	acount := int32(count)
	mu := &sync.Once{}
	lg := s.WithName("FetchWrker")
	workFunc := func() {
		for ids := range idsChan {
			if ids == nil {
				lg.Info("close fetch worker ")
				return
			}
			retrived := false
		next:
			for _, p := range s.GetPeers() {
				peer := p
				lg.Info("send fetch request %v of %v to Peer: %v ", reflect.TypeOf(ids), peer.String())
				ch, _ := reqFactory(s, peer, ids)
				timeout := time.After(s.GetTimeout())
				select {
				case <-timeout:
					lg.Error("fetch request to %v on %v timed out", peer.String(), ids)
				case v := <-ch:
					if v != nil {
						retrived = true
						lg.Info("Peer: %v responded %v to fetch request ", peer.String(), v)
						output <- fetchJob{ids: ids, items: v}
						break next
					}
					lg.Info("next peer")
				}
			}
			if !retrived {
				output <- fetchJob{ids: ids, items: nil}
			}
		}
	}

	return worker{Logger: lg, Once: mu, WaitGroup: &sync.WaitGroup{}, workCount: &acount, output: output, work: workFunc}
}

func NewBlockhWorker(s WorkerInfra, count int, reqFactory BlockRequestFactory, idsChan chan types.Hash32) worker {
	output := make(chan interface{}, count)
	acount := int32(count)
	mu := &sync.Once{}
	lg := s.WithName("BlockWrker")
	workFunc := func() {
		for id := range idsChan {
			lg.Info("id %v out of chan ", id)
			retrived := false
		next:
			for _, p := range s.GetPeers() {
				peer := p
				lg.Info("send fetch block request to Peer: %v id: %v", peer.String(), id)
				ch, _ := reqFactory(s, peer, id)
				timeout := time.After(s.GetTimeout())
				select {
				case <-timeout:
					lg.Error("fetch request to %v on %v timed out", peer.String(), id)
				case v := <-ch:
					if v != nil {
						retrived = true
						lg.Info("Peer: %v responded %v to fetch request ", peer.String(), v)
						output <- blockJob{id: id, item: v.(*types.Block)}
						break next
					}
					lg.Info("next peer")
				}
			}
			if !retrived {
				output <- blockJob{id: id, item: nil}
			}
		}
	}

	return worker{Logger: lg, Once: mu, WaitGroup: &sync.WaitGroup{}, workCount: &acount, output: output, work: workFunc}
}
