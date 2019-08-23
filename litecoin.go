// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import zmq "github.com/pebbe/zmq4"

type litecoinHandler struct {
	*bitcoinHandler
}

func newLitecoinHandler(name string, conf currencyConfig, pubIPv4 *zmq.Socket, pubIPv6 *zmq.Socket) *litecoinHandler {
	return &litecoinHandler{newBitcoinHandler(name, conf, pubIPv4, pubIPv6)}
}
