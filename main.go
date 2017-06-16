package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/bitmark-inc/logger"
	"github.com/hashicorp/hcl"
)

var (
	cfg config
)

type currencyConfig struct {
	URL              string `hcl:"url"`
	SubEndpoint      string `hcl:"sub_endpoint"`
	PubEndpoint      string `hcl:"pub_endpoint"`
	RepEndpoint      string `hcl:"rep_endpoint"`
	CachedBlockCount int    `hcl:"cached_block_count"`
}

type loggerConfig struct {
	File   string `hcl:"file"`
	Size   int    `hcl:"size"`
	Number int    `hcl:"number"`
}

type config struct {
	Bitcoin  currencyConfig
	Litecoin currencyConfig
	Logger   loggerConfig
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
}

func main() {
	bitcoin := newBitcoinHandler("bitcoin", cfg.Bitcoin)
	bitcoin.Run()

	litecoin := newLitecoinHandler("litecoin", cfg.Litecoin)
	litecoin.Run()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
