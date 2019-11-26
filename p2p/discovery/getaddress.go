package discovery

import (
	"errors"
	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/p2p/node"
	"github.com/spacemeshos/go-spacemesh/p2p/p2pcrypto"
	"github.com/spacemeshos/go-spacemesh/p2p/server"
	"time"
)

// todo : calculate real udp max message size

func (p *protocol) newGetAddressesRequestHandler() func(msg server.Message) []byte {
	return func(msg server.Message) []byte {
		p.logger.Debug("Got a get_address request from %v", msg.Sender().String())

		// If we don't know who is that peer (a.k.a first time we hear from this address)
		// we must ensure that he's indeed listening on that address = check last pong
		ka, err := p.table.LookupKnownAddress(msg.Sender())
		if err != nil {
			p.logger.Error("Error looking up message sender (GetAddress) Peer: %v", msg.Sender())
			return nil
		}
		// Check if we've pinged this peer recently enough
		// Should we attempt to send a ping here? It shouldn't be necessary, since the node
		// requesting addresses should have pinged us first, and we should have already sent
		// a ping in response.
		if ka.NeedsPing() {
			p.logger.Warning("Failed ping check (GetAddress) Peer: %v", msg.Sender())
			return nil
			//if err := p.Ping(msg.Sender()); err != nil {
			//	p.logger.Error("Error pinging peer (GetAddress): %v", msg.Sender())
			//	return nil
			//}
		}
		p.logger.Debug("Passed ping check, recently pinged (GetAddress) Peer: %v", msg.Sender())

		results := p.table.AddressCache()
		// remove the sender from the list
		for i, addr := range results {
			if addr.PublicKey() == msg.Sender() {
				results[i] = results[len(results)-1]
				results = results[:len(results)-1]
				break
			}
		}

		//todo: limit results to message size
		//todo: what to do if we have no addresses?
		resp, err := types.InterfaceToBytes(results)

		if err != nil {
			p.logger.Error("Error marshaling response message (GetAddress)")
			return nil
		}

		p.logger.Debug("responding a get_address request from %v", msg.Sender().String())
		return resp
	}
}

// GetAddresses Send a single find node request to a remote node
func (p *protocol) GetAddresses(server p2pcrypto.PublicKey) ([]*node.NodeInfo, error) {
	start := time.Now()
	var err error

	// response handler
	ch := make(chan []*node.NodeInfo)
	resHandler := func(msg []byte) {
		defer close(ch)
		nodes := make([]*node.NodeInfo, 0, getAddrMax)
		err := types.BytesToInterface(msg, &nodes)
		//todo: check that we're not pass max results ?
		if err != nil {
			p.logger.Warning("could not deserialize bytes to NodeInfo, skipping packet err=", err)
			return
		}

		if len(nodes) > getAddrMax {
			p.logger.Warning("addresses response from %v size is too large, ignoring. got: %v, expected: < %v", server.String(), len(nodes), getAddrMax)
			return
		}

		ch <- nodes
	}

	err = p.msgServer.SendRequest(GET_ADDRESSES, []byte(""), server, resHandler)

	if err != nil {
		return nil, err
	}

	timeout := time.NewTimer(MessageTimeout)
	select {
	case nodes := <-ch:
		if nodes == nil {
			return nil, errors.New("empty result set")
		}
		p.logger.With().Debug("getaddress_time_to_recv", log.String("from", server.String()), log.Duration("time_elapsed", time.Now().Sub(start)))
		return nodes, nil
	case <-timeout.C:
		return nil, errors.New("request timed out")
	}
}
