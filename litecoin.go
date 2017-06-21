package main

import zmq "github.com/pebbe/zmq4"

type litecoinHandler struct {
	*bitcoinHandler
}

func newLitecoinHandler(name string, conf currencyConfig, pub *zmq.Socket) *litecoinHandler {
	return &litecoinHandler{newBitcoinHandler(name, conf, pub)}
}
