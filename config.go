package main

import (
	"github.com/bitmark-inc/logger"
)

type currencyConfig struct {
	URL              string `gluamapper:"url" json:"url"`
	SubEndpoint      string `gluamapper:"sub_endpoint" json:"sub_endpoint"`
	CachedBlockCount int    `gluamapper:"cached_block_count" json:"cached_block_count"`
}

type config struct {
	DataDirectory string `gluamapper:"data_directory" json:"data_directory"`
	PubEndpoint   string `gluamapper:"pub_endpoint" json:"pub_endpoint"`
	RepEndpoint   string `gluamapper:"rep_endpoint" json:"rep_endpoint"`
	Currency      struct {
		Bitcoin  currencyConfig
		Litecoin currencyConfig
	} `gluamapper:"currency" json:"currency"`
	Logging logger.Configuration `gluamapper:"logging" json:"loggin"`
}
