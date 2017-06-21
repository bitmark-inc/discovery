package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
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
	handleTxQuery(query string) (string, interface{})
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
	handlers["bitcoin"] = newBitcoinHandler("bitcoin", cfg.Currency.Bitcoin, pub)
	handlers["litecoin"] = newLitecoinHandler("litecoin", cfg.Currency.Litecoin, pub)
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
		query := string(msg[1])
		status, dat := handlers[currency].handleTxQuery(query)
		rep.SendMessage(status, dat)
	}
}
