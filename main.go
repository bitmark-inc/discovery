package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
)

var (
	cfg config
)

type currencyConfig struct {
	URL              string `mapstructure:"url"`
	SubEndpoint      string `mapstructure:"sub_endpoint"`
	PubEndpoint      string `mapstructure:"sub_endpoint"`
	RepEndpoint      string `mapstructure:"rep_endpoint"`
	CachedBlockCount int    `mapstructure:"cached_block_count"`
}

type config struct {
	Bitcoin  currencyConfig
	Litecoin currencyConfig
}

func init() {
	viper.SetConfigName("conf")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("read conf failed: %s", err))
	}
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("parse conf failed: %s", err))
	}
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
