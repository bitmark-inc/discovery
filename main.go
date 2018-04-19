// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/bitmark-inc/logger"
	"github.com/hashicorp/hcl"
	zmq "github.com/pebbe/zmq4"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

var (
	cfg      config
	log      *logger.L
	pub      *zmq.Socket
	rep      *zmq.Socket
	handlers map[string]cryptoCurrencyHandler
)

type currencyConfig struct {
	URL              string `hcl:"url"`
	SubEndpoint      string `hcl:"sub_endpoint"`
	CachedBlockCount int    `hcl:"cached_block_count"`
}

type config struct {
	PubEndpoint string `hcl:"pub_endpoint"`
	RepEndpoint string `hcl:"rep_endpoint"`
	Currency    struct {
		Bitcoin  currencyConfig
		Litecoin currencyConfig
	} `hcl:"currency"`
	Logging logger.Configuration `hcl:"logging"`
}

type cryptoCurrencyHandler interface {
	rescanRecentBlocks(wg *sync.WaitGroup)
	handleTxQuery(ts int64) interface{}
	listenBlockchain()
}

func init() {
	var path string
	flag.StringVar(&path, "conf", "", "Specify configuration file")
	flag.Parse()

	dat, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read conf file failed: %s", err))
	}

	if err = hcl.Unmarshal(dat, &cfg); nil != err {
		panic(fmt.Sprintf("parse conf file failed: %s", err))
	}

	if err = logger.Initialise(cfg.Logging); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	log = logger.New("discovery")

	pub, err = zmq.NewSocket(zmq.PUB)
	if err != nil {
		panic(err)
	}
	pub.SetIpv6(true)

	err = pub.Bind(cfg.PubEndpoint)
	if err != nil {
		panic(err)
	}

	rep, err = zmq.NewSocket(zmq.REP)
	if err != nil {
		panic(err)
	}
	rep.SetIpv6(true)

	err = rep.Bind(cfg.RepEndpoint)
	if err != nil {
		panic(err)
	}
	handlers = make(map[string]cryptoCurrencyHandler)
	handlers["BTC"] = newBitcoinHandler("BTC", cfg.Currency.Bitcoin, pub)
	handlers["LTC"] = newLitecoinHandler("LTC", cfg.Currency.Litecoin, pub)
}

func main() {
	for _, handler := range handlers {
		go handler.listenBlockchain()
	}

	go serveRequest()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	logger.Finalise()
}

type paymentTxs struct {
	Bitcoin  interface{} `json:"bitcoin"`
	Litecoin interface{} `json:"litecoin"`
}

func serveRequest() {
	var wg sync.WaitGroup

	for _, h := range handlers {
		wg.Add(1)
		go h.rescanRecentBlocks(&wg)
	}
	wg.Wait()

	log.Info("start to serve requests")
serve_requests:
	for {
		msg, err := rep.RecvMessageBytes(0)
		if nil != err {
			log.Errorf("failed to receive request message: %s", err)
			rep.SendMessage("ERROR", err)
			continue serve_requests
		}

		currency := string(msg[0])
		ts, err := strconv.ParseInt(string(msg[1]), 10, 64)
		if err != nil {
			rep.SendMessage("ERROR", errors.New("incorrect parameter"))
			continue serve_requests
		}

		txs := handlers[currency].handleTxQuery(ts)
		dat, _ := json.Marshal(&txs)
		rep.SendMessage("OK", dat)
	}
}
