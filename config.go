package main

import (
	"github.com/bitmark-inc/logger"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
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

// ParseConfigurationFile parse lua config
func ParseConfigurationFile(fileName string, config interface{}) error {
	L := lua.NewState()
	defer L.Close()

	L.OpenLibs()

	// create the global "arg" table
	// arg[0] = config file
	arg := &lua.LTable{}
	arg.Insert(0, lua.LString(fileName))
	L.SetGlobal("arg", arg)

	// execute configuration
	if err := L.DoFile(fileName); err != nil {
		return err
	}

	mapperOption := gluamapper.Option{
		NameFunc: func(s string) string {
			return s
		},
		TagName: "gluamapper",
	}
	mapper := gluamapper.Mapper{Option: mapperOption}
	if err := mapper.Map(L.Get(L.GetTop()).(*lua.LTable), config); err != nil {
		return err
	}

	return nil
}
