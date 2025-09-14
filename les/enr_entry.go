// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package les

import (
	"github.com/microstack-tech/parallax/core/forkid"
	"github.com/microstack-tech/parallax/p2p/dnsdisc"
	"github.com/microstack-tech/parallax/p2p/enode"
	"github.com/microstack-tech/parallax/rlp"
)

// lpsEntry is the "lps" LPS entry. This is set for LPS servers only.
type lpsEntry struct {
	// Ignore additional fields (for forward compatibility).
	VfxVersion uint
	Rest       []rlp.RawValue `rlp:"tail"`
}

func (lpsEntry) ENRKey() string { return "les" }

// ethEntry is the "eth" ENR entry. This is redeclared here to avoid depending on package eth.
type ethEntry struct {
	ForkID forkid.ID
	Tail   []rlp.RawValue `rlp:"tail"`
}

func (ethEntry) ENRKey() string { return "eth" }

// setupDiscovery creates the node discovery source for the eth protocol.
func (eth *LightParallax) setupDiscovery() (enode.Iterator, error) {
	it := enode.NewFairMix(0)

	// Enable DNS discovery.
	if len(eth.config.EthDiscoveryURLs) != 0 {
		client := dnsdisc.NewClient(dnsdisc.Config{})
		dns, err := client.NewIterator(eth.config.EthDiscoveryURLs...)
		if err != nil {
			return nil, err
		}
		it.AddSource(dns)
	}

	// Enable DHT.
	if eth.udpEnabled {
		it.AddSource(eth.p2pServer.DiscV5.RandomNodes())
	}

	forkFilter := forkid.NewFilter(eth.blockchain)
	iterator := enode.Filter(it, func(n *enode.Node) bool { return nodeIsServer(forkFilter, n) })
	return iterator, nil
}

// nodeIsServer checks whether n is an LPS server node.
func nodeIsServer(forkFilter forkid.Filter, n *enode.Node) bool {
	var les lpsEntry
	var eth ethEntry
	return n.Load(&les) == nil && n.Load(&eth) == nil && forkFilter(eth.ForkID) == nil
}
