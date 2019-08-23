package main

import (
	"github.com/bitmark-inc/logger"
)

type currencyConfig struct {
	URL              string `gluamapper:"url" json:"url"`
	SubEndpoint      string `gluamapper:"sub_endpoint" json:"sub_endpoint"`
	CachedBlockCount int    `gluamapper:"cached_block_count" json:"cached_block_count"`
}

type ipv4v6 struct {
	IPv4 string `gluamapper:"ipv4" json:"ipv4"`
	IPv6 string `gluamapper:"ipv6" json:"ipv6"`
}

type currencyPair struct {
	Bitcoin  currencyConfig `gluamapper:"bitcoin" json:"bitcoin"`
	Litecoin currencyConfig `gluamapper:"litecoin" json:"litecoin"`
}

type config struct {
	DataDirectory string               `gluamapper:"data_directory" json:"data_directory"`
	PubEndpoint   ipv4v6               `gluamapper:"pub_endpoint" json:"pub_endpoint"`
	RepEndpoint   ipv4v6               `gluamapper:"rep_endpoint" json:"rep_endpoint"`
	Currency      currencyPair         `gluamapper:"currency" json:"currency"`
	Logging       logger.Configuration `gluamapper:"logging" json:"logging"`
}
