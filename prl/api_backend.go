// Copyright 2015 The go-ethereum Authors
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

package prl

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/microstack-tech/parallax"
	"github.com/microstack-tech/parallax/accounts"
	"github.com/microstack-tech/parallax/common"
	"github.com/microstack-tech/parallax/consensus"
	"github.com/microstack-tech/parallax/core"
	"github.com/microstack-tech/parallax/core/bloombits"
	"github.com/microstack-tech/parallax/core/rawdb"
	"github.com/microstack-tech/parallax/core/state"
	"github.com/microstack-tech/parallax/core/types"
	"github.com/microstack-tech/parallax/core/vm"
	"github.com/microstack-tech/parallax/event"
	"github.com/microstack-tech/parallax/miner"
	"github.com/microstack-tech/parallax/params"
	"github.com/microstack-tech/parallax/prl/gasprice"
	"github.com/microstack-tech/parallax/prldb"
	"github.com/microstack-tech/parallax/rpc"
)

// PrlAPIBackend implements prlapi.Backend for full nodes
type PrlAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	eth                 *Parallax
	gpo                 *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *PrlAPIBackend) ChainConfig() *params.ChainConfig {
	return b.eth.blockchain.Config()
}

func (b *PrlAPIBackend) CurrentBlock() *types.Block {
	return b.eth.blockchain.CurrentBlock()
}

func (b *PrlAPIBackend) SetHead(number uint64) {
	b.eth.handler.downloader.Cancel()
	b.eth.blockchain.SetHead(number)
}

func (b *PrlAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.eth.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.eth.blockchain.CurrentBlock().Header(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		return b.eth.blockchain.CurrentFinalizedBlock().Header(), nil
	}
	return b.eth.blockchain.GetHeaderByNumber(uint64(number)), nil
}

func (b *PrlAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.eth.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.eth.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *PrlAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.eth.blockchain.GetHeaderByHash(hash), nil
}

func (b *PrlAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.eth.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.eth.blockchain.CurrentBlock(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		return b.eth.blockchain.CurrentFinalizedBlock(), nil
	}
	return b.eth.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *PrlAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.eth.blockchain.GetBlockByHash(hash), nil
}

func (b *PrlAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.eth.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.eth.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.eth.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *PrlAPIBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	return b.eth.miner.PendingBlockAndReceipts()
}

func (b *PrlAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, state := b.eth.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.eth.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *PrlAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.eth.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.eth.BlockChain().StateAt(header.Root)
		return stateDb, header, err
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *PrlAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.eth.blockchain.GetReceiptsByHash(hash), nil
}

func (b *PrlAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	db := b.eth.ChainDb()
	number := rawdb.ReadHeaderNumber(db, hash)
	if number == nil {
		return nil, fmt.Errorf("failed to get block number for hash %#x", hash)
	}
	logs := rawdb.ReadLogs(db, hash, *number, b.eth.blockchain.Config())
	if logs == nil {
		return nil, fmt.Errorf("failed to get logs for block #%d (0x%s)", *number, hash.TerminalString())
	}
	return logs, nil
}

func (b *PrlAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	if header := b.eth.blockchain.GetHeaderByHash(hash); header != nil {
		return b.eth.blockchain.GetTd(hash, header.Number.Uint64())
	}
	return nil
}

func (b *PrlAPIBackend) GetPVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config) (*vm.PVM, func() error, error) {
	vmError := func() error { return nil }
	if vmConfig == nil {
		vmConfig = b.eth.blockchain.GetVMConfig()
	}
	txContext := core.NewPVMTxContext(msg)
	context := core.NewPVMBlockContext(header, b.eth.BlockChain(), nil)
	return vm.NewPVM(context, txContext, state, b.eth.blockchain.Config(), *vmConfig), vmError, nil
}

func (b *PrlAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *PrlAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.eth.miner.SubscribePendingLogs(ch)
}

func (b *PrlAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainEvent(ch)
}

func (b *PrlAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *PrlAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.eth.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *PrlAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.eth.BlockChain().SubscribeLogsEvent(ch)
}

func (b *PrlAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.eth.txPool.AddLocal(signedTx)
}

func (b *PrlAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending := b.eth.txPool.Pending(false)
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *PrlAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.eth.txPool.Get(hash)
}

func (b *PrlAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.eth.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *PrlAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.eth.txPool.Nonce(addr), nil
}

func (b *PrlAPIBackend) Stats() (pending int, queued int) {
	return b.eth.txPool.Stats()
}

func (b *PrlAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.eth.TxPool().Content()
}

func (b *PrlAPIBackend) TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	return b.eth.TxPool().ContentFrom(addr)
}

func (b *PrlAPIBackend) TxPool() *core.TxPool {
	return b.eth.TxPool()
}

func (b *PrlAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.eth.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *PrlAPIBackend) SyncProgress() parallax.SyncProgress {
	return b.eth.Downloader().Progress()
}

func (b *PrlAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestTipCap(ctx)
}

func (b *PrlAPIBackend) FeeHistory(ctx context.Context, blockCount int, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock *big.Int, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *PrlAPIBackend) ChainDb() prldb.Database {
	return b.eth.ChainDb()
}

func (b *PrlAPIBackend) EventMux() *event.TypeMux {
	return b.eth.EventMux()
}

func (b *PrlAPIBackend) AccountManager() *accounts.Manager {
	return b.eth.AccountManager()
}

func (b *PrlAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *PrlAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *PrlAPIBackend) RPCGasCap() uint64 {
	return b.eth.config.RPCGasCap
}

func (b *PrlAPIBackend) RPCPVMTimeout() time.Duration {
	return b.eth.config.RPCPVMTimeout
}

func (b *PrlAPIBackend) RPCTxFeeCap() float64 {
	return b.eth.config.RPCTxFeeCap
}

func (b *PrlAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.eth.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *PrlAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.eth.bloomRequests)
	}
}

func (b *PrlAPIBackend) Engine() consensus.Engine {
	return b.eth.engine
}

func (b *PrlAPIBackend) CurrentHeader() *types.Header {
	return b.eth.blockchain.CurrentHeader()
}

func (b *PrlAPIBackend) Miner() *miner.Miner {
	return b.eth.Miner()
}

func (b *PrlAPIBackend) StartMining(threads int) error {
	return b.eth.StartMining(threads)
}

func (b *PrlAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, checkLive, preferDisk bool) (*state.StateDB, error) {
	return b.eth.StateAtBlock(block, reexec, base, checkLive, preferDisk)
}

func (b *PrlAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, error) {
	return b.eth.stateAtTransaction(block, txIndex, reexec)
}
