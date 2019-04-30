// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	zmq "github.com/pebbe/zmq4"

	lua "github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/logger"
)

var (
	cfg      config
	log      *logger.L
	pub      *zmq.Socket
	rep      *zmq.Socket
	handlers map[string]cryptoCurrencyHandler
)

type cryptoCurrencyHandler interface {
	rescanRecentBlocks(wg *sync.WaitGroup)
	handleTxQuery(ts int64) interface{}
	listenBlockchain()
}

func init() {
	var path string
	flag.StringVar(&path, "conf", "", "Specify configuration file")
	flag.Parse()
	if err := lua.ParseConfigurationFile(path, &cfg); err != nil {
		panic(fmt.Sprintf("config file read failed: %s", err))
	}
	if err := logger.Initialise(cfg.Logging); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	log = logger.New("discovery")
	log.Info(fmt.Sprintf("DataDirectory:%s", cfg.DataDirectory))
	log.Info(fmt.Sprintf("PubEndpoint:%s", cfg.PubEndpoint))
	log.Info(fmt.Sprintf("RepEndpoint:%s", cfg.RepEndpoint))
	log.Info(fmt.Sprintf("Bitcoin URL:%s CachedBlockCount:%d SubEndpoint:%s", cfg.Currency.Bitcoin.URL, cfg.Currency.Bitcoin.CachedBlockCount, cfg.Currency.Bitcoin.SubEndpoint))
	log.Info(fmt.Sprintf("Litecoin URL:%s CachedBlockCount:%d SubEndpoint:%s", cfg.Currency.Litecoin.URL, cfg.Currency.Litecoin.CachedBlockCount, cfg.Currency.Litecoin.SubEndpoint))
	log.Info(fmt.Sprintf("LogDir:%s LogFile:%s", cfg.Logging.Directory, cfg.Logging.File))

	pub, err := zmq.NewSocket(zmq.PUB)
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
