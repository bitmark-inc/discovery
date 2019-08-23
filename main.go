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
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"

	zmq "github.com/pebbe/zmq4"

	lua "github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/logger"
)

type cryptoCurrencyHandler interface {
	rescanRecentBlocks(wg *sync.WaitGroup)
	handleTxQuery(ts int64) interface{}
	listenBlockchain()
}

func main() {
	var cfg config
	var configurationFile string
	flag.StringVar(&configurationFile, "conf", "", "Specify configuration file")
	flag.Parse()
	if err := lua.ParseConfigurationFile(configurationFile, &cfg); err != nil {
		panic(fmt.Sprintf("config file read failed: %s", err))
	}

	if !strings.HasPrefix(cfg.Logging.Directory, "/") {
		cfg.Logging.Directory = path.Join(cfg.DataDirectory, cfg.Logging.Directory)
	}

	if err := logger.Initialise(cfg.Logging); err != nil {
		panic(fmt.Sprintf("logger initialization failed: %s", err))
	}
	log := logger.New("discovery")
	log.Infof("DataDirectory: %s", cfg.DataDirectory)
	log.Infof("PubEndpoint IPv4: %s", cfg.PubEndpoint.IPv4)
	log.Infof("PubEndpoint IPv6: %s", cfg.PubEndpoint.IPv6)
	log.Infof("RepEndpoint IPv4: %s", cfg.RepEndpoint.IPv4)
	log.Infof("RepEndpoint IPv6: %s", cfg.RepEndpoint.IPv6)
	log.Infof("Bitcoin URL: %s  CachedBlockCount: %d  SubEndpoint: %s", cfg.Currency.Bitcoin.URL, cfg.Currency.Bitcoin.CachedBlockCount, cfg.Currency.Bitcoin.SubEndpoint)
	log.Infof("Litecoin URL: %s  CachedBlockCount: %d  SubEndpoint: %s", cfg.Currency.Litecoin.URL, cfg.Currency.Litecoin.CachedBlockCount, cfg.Currency.Litecoin.SubEndpoint)
	log.Infof("LogDir: %s  LogFile: %s", cfg.Logging.Directory, cfg.Logging.File)

	pubIPv4, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		panic(err)
	}

	pubIPv6, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		panic(err)
	}
	pubIPv6.SetIpv6(true)

	err = pubIPv4.Bind(cfg.PubEndpoint.IPv4)
	if err != nil {
		panic(err)
	}
	err = pubIPv6.Bind(cfg.PubEndpoint.IPv6)
	if err != nil {
		panic(err)
	}

	repIPv4, err := zmq.NewSocket(zmq.REP)
	if err != nil {
		panic(err)
	}

	repIPv6, err := zmq.NewSocket(zmq.REP)
	if err != nil {
		panic(err)
	}
	repIPv6.SetIpv6(true)

	err = repIPv4.Bind(cfg.RepEndpoint.IPv4)
	if err != nil {
		panic(err)
	}
	err = repIPv6.Bind(cfg.RepEndpoint.IPv6)
	if err != nil {
		panic(err)
	}

	handlers := make(map[string]cryptoCurrencyHandler)
	handlers["BTC"] = newBitcoinHandler("BTC", cfg.Currency.Bitcoin, pubIPv4, pubIPv6)
	handlers["LTC"] = newLitecoinHandler("LTC", cfg.Currency.Litecoin, pubIPv4, pubIPv6)

	// scan all chains
	for _, handler := range handlers {
		go handler.listenBlockchain()
	}

	var wg sync.WaitGroup

	for _, h := range handlers {
		wg.Add(1)
		go h.rescanRecentBlocks(&wg)
	}
	wg.Wait()

	// start request servers
	go serveRequest(repIPv4, handlers, logger.New("rep-ipv4"))
	go serveRequest(repIPv6, handlers, logger.New("rep-ipv6"))

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	logger.Finalise()
}

type paymentTxs struct {
	Bitcoin  interface{} `json:"bitcoin"`
	Litecoin interface{} `json:"litecoin"`
}

func serveRequest(rep *zmq.Socket, handlers map[string]cryptoCurrencyHandler, log *logger.L) {

	if nil == rep {
		log.Error("rep socket is nil")
		return
	}
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

		log.Debugf("currency: %s  ts: %d", currency, ts)

		txs := handlers[currency].handleTxQuery(ts)
		dat, _ := json.Marshal(&txs)
		rep.SendMessage("OK", dat)
	}
}
