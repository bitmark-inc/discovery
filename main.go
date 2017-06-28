package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/bitmark-inc/logger"
	"github.com/hashicorp/hcl"
	zmq "github.com/pebbe/zmq4"
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
	}
	Logger struct {
		File   string `hcl:"file"`
		Size   int    `hcl:"size"`
		Number int    `hcl:"number"`
	}
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

	logger.Initialise(cfg.Logger.File, cfg.Logger.Size, cfg.Logger.Number)
	log = logger.New("discovery")
	log.ChangeLevel("info")

	pub, err = zmq.NewSocket(zmq.PUB)
	if err != nil {
		panic(err)
	}
	pub.Bind(cfg.PubEndpoint)

	rep, err = zmq.NewSocket(zmq.REP)
	if err != nil {
		panic(err)
	}
	rep.Bind(cfg.RepEndpoint)

	handlers = make(map[string]cryptoCurrencyHandler)
	handlers["btc"] = newBitcoinHandler("btc", cfg.Currency.Bitcoin, pub)
	handlers["ltc"] = newLitecoinHandler("ltc", cfg.Currency.Litecoin, pub)
}

func main() {
	for _, handler := range handlers {
		go handler.listenBlockchain()
	}

	go serveRequest()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
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
	for {
		msg, err := rep.RecvMessageBytes(0)
		if nil != err {
			log.Errorf("failed to receive request message: %s", err)
			rep.SendMessage("ERROR", err)
			continue
		}

		currency := string(msg[0])
		ts, err := strconv.ParseInt(string(msg[1]), 10, 64)
		if err != nil {
			rep.SendMessage("ERROR", errors.New("incorrect parameter"))
		}

		txs := handlers[currency].handleTxQuery(ts)
		dat, _ := json.Marshal(&txs)
		rep.SendMessage("OK", dat)
	}
}
