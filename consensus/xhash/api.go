// Copyright 2018 The go-ethereum Authors
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

package xhash

import (
	"errors"
	"math/big"

	"github.com/microstack-tech/parallax/common"
	"github.com/microstack-tech/parallax/common/hexutil"
	"github.com/microstack-tech/parallax/core/types"
)

var errXHashStopped = errors.New("xhash stopped")

// API exposes xhash related methods for the RPC interface.
type API struct {
	xhash *XHash
}

// GetWork returns a work package for external miner.
//
// The work package consists of 3 strings:
//
//	result[0] - 32 bytes hex encoded current block header pow-hash
//	result[1] - 32 bytes hex encoded seed hash used for DAG
//	result[2] - 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
//	result[3] - hex encoded block number
func (api *API) GetWork() ([4]string, error) {
	if api.xhash.remote == nil {
		return [4]string{}, errors.New("not supported")
	}

	var (
		workCh = make(chan [4]string, 1)
		errc   = make(chan error, 1)
	)
	select {
	case api.xhash.remote.fetchWorkCh <- &sealWork{errc: errc, res: workCh}:
	case <-api.xhash.remote.exitCh:
		return [4]string{}, errXHashStopped
	}
	select {
	case work := <-workCh:
		return work, nil
	case err := <-errc:
		return [4]string{}, err
	}
}

// SubmitWork can be used by external miner to submit their POW solution.
// It returns an indication if the work was accepted.
// Note either an invalid solution, a stale work a non-existent work will return false.
func (api *API) SubmitWork(nonce types.BlockNonce, hash, digest common.Hash) bool {
	if api.xhash.remote == nil {
		return false
	}

	errc := make(chan error, 1)
	select {
	case api.xhash.remote.submitWorkCh <- &mineResult{
		nonce:     nonce,
		mixDigest: digest,
		hash:      hash,
		errc:      errc,
	}:
	case <-api.xhash.remote.exitCh:
		return false
	}
	err := <-errc
	return err == nil
}

// SubmitHashrate can be used for remote miners to submit their hash rate.
// This enables the node to report the combined hash rate of all miners
// which submit work through this node.
//
// It accepts the miner hash rate and an identifier which must be unique
// between nodes.
func (api *API) SubmitHashrate(rate hexutil.Uint64, id common.Hash) bool {
	if api.xhash.remote == nil {
		return false
	}

	done := make(chan struct{}, 1)
	select {
	case api.xhash.remote.submitRateCh <- &hashrate{done: done, rate: uint64(rate), id: id}:
	case <-api.xhash.remote.exitCh:
		return false
	}

	// Block until hash rate submitted successfully.
	<-done
	return true
}

// GetHashrate returns the current hashrate for local CPU miner and remote miner.
func (api *API) GetHashrate() uint64 {
	return uint64(api.xhash.Hashrate())
}

func (api *API) GetCumulativeEmissions(blockNumber hexutil.Uint64) *big.Int {
	emissions := big.NewInt(0)
	number := uint64(blockNumber)

	for i := uint64(0); i < number; i++ {
		reward := calcBlockReward(number)
		emissions = emissions.Add(emissions, reward)
	}
	return emissions
}
