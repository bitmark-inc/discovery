package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
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
	Hash string        `json:"hash"`
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

type pastBitcoinPaymentTransactions struct {
	Transactions []bitcoinTransaction `json:"txs"`
}

type bitcoinHandler struct {
	sync.RWMutex
	name             string
	logger           *logger.L
	fetcher          *fetcher
	sub              *zmq.Socket
	pub              *zmq.Socket
	rep              *zmq.Socket
	cachedBlockCount int
	cachedBlocks     []bitcoinBlock
}

func newBitcoinHandler(name string, conf currencyConfig) *bitcoinHandler {
	pub, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		panic(err)
	}
	pub.Bind(conf.PubEndpoint)

	sub, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		panic(err)
	}
	sub.Connect(conf.SubEndpoint)
	sub.SetSubscribe("hashtx")
	sub.SetSubscribe("hashblock")

	rep, err := zmq.NewSocket(zmq.REP)
	if err != nil {
		panic(err)
	}
	rep.Bind(conf.RepEndpoint)

	logger := logger.New(name)
	logger.ChangeLevel("info")

	return &bitcoinHandler{
		name: name, logger: logger, fetcher: &fetcher{conf.URL},
		sub: sub, pub: pub, rep: rep,
		cachedBlockCount: conf.CachedBlockCount, cachedBlocks: make([]bitcoinBlock, 0),
	}
}

func (b *bitcoinHandler) Run() {
	done := make(chan struct{})
	go b.rescanRecentBlocks(done)
	go b.serveRequest(done)

	go b.listenBlockchain()
}

func (b *bitcoinHandler) rescanRecentBlocks(done chan<- struct{}) {
	b.logger.Info("start rescaning")

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

	b.logger.Info("end rescaning")

	done <- struct{}{}
}

func (b *bitcoinHandler) serveRequest(done <-chan struct{}) {
	<-done

	for {
		msg, err := b.rep.RecvMessageBytes(0)
		if nil != err {
			b.logger.Errorf("zmq recv error: %s", err)
			continue
		}

		ts, err := strconv.ParseInt(string(msg[0]), 10, 64)
		if err != nil {
			b.rep.SendMessage("ERROR", err.Error())
		}

		txs := make([]bitcoinTransaction, 0)
		b.RLock()
		for _, block := range b.cachedBlocks {
			if ts > block.Time {
				break
			}

			for _, tx := range block.Tx {
				if isPaymentTransaction(&tx) {
					txs = append(txs, tx)
				}
			}
		}
		b.RUnlock()

		pastTXs := pastBitcoinPaymentTransactions{txs}
		bytes, _ := json.Marshal(pastTXs)
		b.rep.SendMessage("OK", bytes)
	}
}

func (b *bitcoinHandler) listenBlockchain() {
	for {
		msg, err := b.sub.RecvMessageBytes(0)
		if err != nil {
			b.logger.Errorf("zmq recv error: %s", err)
			continue
		}

		switch topic := string(msg[0]); topic {
		case "hashtx":
			txHash := hex.EncodeToString(msg[1])
			b.logger.Debugf("tx hash received: %v", txHash)
			b.processNewTransaction(txHash)
		case "hashblock":
			blockHash := hex.EncodeToString(msg[1])
			b.logger.Infof("block hash received: %v", blockHash)
			b.processNewBlock(blockHash)
		}
	}
}

func (b *bitcoinHandler) processNewTransaction(txHash string) {
	var tx bitcoinTransaction
	if err := b.fetcher.fetch(fmt.Sprintf("/rest/tx/%s.json", txHash), &tx); err != nil {
		b.logger.Errorf("fetch new tx failed: %s", err)
	}

	if isPaymentTransaction(&tx) {
		data, _ := json.Marshal(tx)
		b.pub.SendMessage("bitcoin", data)
	}
}

func (b *bitcoinHandler) processNewBlock(blockHash string) {
	var block bitcoinBlock
	if err := b.fetcher.fetch(fmt.Sprintf("/rest/block/%s.json", blockHash), &block); err != nil {
		b.logger.Errorf("fetch new block failed: %s", err)
	}

	b.Lock()
	b.cachedBlocks = append(b.cachedBlocks, block)
	b.cachedBlocks = b.cachedBlocks[1:]
	b.Unlock()
}

func isPaymentTransaction(tx *bitcoinTransaction) bool {
	for _, vout := range tx.Vout {
		if bitcoinOPReturnRecordLength == len(vout.ScriptPubKey.Hex) && bitcoinOPReturnHexCode == vout.ScriptPubKey.Hex[0:4] {
			return true
		}
	}
	return false
}
