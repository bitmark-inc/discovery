// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"sync"
)

const (
	bitcoinOPReturnHexCode      = "6a30"
	bitcoinOPReturnPrefixLength = len(bitcoinOPReturnHexCode)
	bitcoinOPReturnRecordLength = bitcoinOPReturnPrefixLength + 2*48
)

type bitcoinScriptPubKey struct {
	Hex       string   `json:"hex"`
	Addresses []string `json:"addresses"`
}

type bitcoinVout struct {
	Value        json.RawMessage     `json:"value"`
	ScriptPubKey bitcoinScriptPubKey `json:"scriptPubKey"`
}

type bitcoinTransaction struct {
	TxID string        `json:"txid"`
	Vout []bitcoinVout `json:"vout"`
}

type bitcoinBlock struct {
	Hash              string               `json:"hash"`
	Tx                []bitcoinTransaction `json:"tx"`
	PreviousBlockhash string               `json:"previousblockhash"`
	Time              int64                `json:"time"`
}

type bitcoinChainInfo struct {
	Bestblockhash string `json:"bestblockhash"`
}

type bitcoinHandler struct {
	sync.RWMutex
	name             string
	log              *logger.L
	fetcher          *fetcher
	sub              *zmq.Socket
	pub              *zmq.Socket
	cachedBlockCount int
	cachedBlocks     []bitcoinBlock
}

func newBitcoinHandler(name string, conf currencyConfig, pub *zmq.Socket) *bitcoinHandler {
	sub, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		panic(err)
	}
	sub.Connect(conf.SubEndpoint)
	sub.SetSubscribe("hashtx")
	sub.SetSubscribe("hashblock")

	log := logger.New(name)
	log.Info("starting…")

	return &bitcoinHandler{
		name:             name,
		log:              log,
		fetcher:          &fetcher{conf.URL},
		sub:              sub,
		pub:              pub,
		cachedBlockCount: conf.CachedBlockCount,
		cachedBlocks:     make([]bitcoinBlock, 0),
	}
}

func (b *bitcoinHandler) rescanRecentBlocks(wg *sync.WaitGroup) {
	defer wg.Done()

	b.log.Info("start rescaning")

	var info bitcoinChainInfo
	b.fetcher.fetch("/rest/chaininfo.json", &info)

	blocks := make([]bitcoinBlock, b.cachedBlockCount)
	blockHash := info.Bestblockhash
	for i := 0; i < b.cachedBlockCount; i++ {
		var block bitcoinBlock
		b.fetcher.fetch(fmt.Sprintf("/rest/block/%s.json", blockHash), &block)
		blocks[b.cachedBlockCount-i-1] = block
		blockHash = block.PreviousBlockhash
	}

	b.Lock()
	b.cachedBlocks = append(blocks, b.cachedBlocks...)
	b.Unlock()

	b.log.Info("end rescaning")
}

func (b *bitcoinHandler) handleTxQuery(ts int64) interface{} {
	b.RLock()
	blocks := b.cachedBlocks
	b.RUnlock()

	txs := make([]bitcoinTransaction, 0)
scan_blocks:
	for _, block := range blocks {
		if block.Time < ts {
			continue scan_blocks
		}

		for _, tx := range block.Tx {
			if isBitcoinPaymentTX(&tx) {
				txs = append(txs, tx)
			}
		}
	}

	return txs
}

func (b *bitcoinHandler) listenBlockchain() {
loop:
	for {
		msg, err := b.sub.RecvMessageBytes(0)
		if err != nil {
			b.log.Errorf("zmq recv error: %s", err)
			continue loop
		}

		switch topic := string(msg[0]); topic {
		case "hashtx":
			txHash := hex.EncodeToString(msg[1])
			b.log.Debugf("tx hash received: %v", txHash)
			b.processNewTx(txHash)
		case "hashblock":
			blockHash := hex.EncodeToString(msg[1])
			b.log.Infof("block hash received: %v", blockHash)
			b.processNewBlock(blockHash)
		}
	}
}

func (b *bitcoinHandler) processNewTx(txHash string) {
	var tx bitcoinTransaction
	if err := b.fetcher.fetch(fmt.Sprintf("/rest/tx/%s.json", txHash), &tx); err != nil {
		b.log.Errorf("fetch new tx failed: %s", err)
	}

	if isBitcoinPaymentTX(&tx) {
		b.log.Infof("payment tx id: %s", tx.TxID)
		data, _ := json.Marshal(tx)
		b.pub.SendMessage(b.name, data)
	}
}

func (b *bitcoinHandler) processNewBlock(blockHash string) {
	var block bitcoinBlock
	if err := b.fetcher.fetch(fmt.Sprintf("/rest/block/%s.json", blockHash), &block); err != nil {
		b.log.Errorf("fetch new block failed: %s", err)
	}

	b.Lock()
	b.cachedBlocks = append(b.cachedBlocks, block)
	b.cachedBlocks = b.cachedBlocks[1:]
	b.Unlock()
}

func isBitcoinPaymentTX(tx *bitcoinTransaction) bool {
	for _, vout := range tx.Vout {
		if bitcoinOPReturnRecordLength == len(vout.ScriptPubKey.Hex) && bitcoinOPReturnHexCode == vout.ScriptPubKey.Hex[0:4] {
			return true
		}
	}
	return false
}
