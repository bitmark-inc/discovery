// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	zmq "github.com/pebbe/zmq4"
	"os"
	"strings"
)

func main() {

	command := "help"
	if len(os.Args) >= 2 {
		command = os.Args[1]
	}
	switch command {
	case "listen":
		currency := ""
		addr := ""
		subcmd := flag.NewFlagSet("listen", flag.ExitOnError)
		subcmd.StringVar(&currency, "currency", "btc", "Specify the currency (possible values: btc, ltc)")
		subcmd.StringVar(&addr, "addr", "", "Specify the zeromq socket address of discovery proxy")
		if len(os.Args) < 2 {
			subcmd.PrintDefaults()
			os.Exit(1)
		}
		subcmd.Parse(os.Args[2:])
		if "" == currency || "" == addr {
			subcmd.PrintDefaults()
			os.Exit(1)
		}
		currency = strings.ToUpper(currency)
		listen(currency, addr)
	case "query":
		currency := ""
		addr := ""
		timestamp := 0
		subcmd := flag.NewFlagSet("query", flag.ExitOnError)
		subcmd.StringVar(&currency, "currency", "btc", "Specify the currency (possible values: btc, ltc)")
		subcmd.StringVar(&addr, "addr", "", "Specify the zeromq socket address of discovery proxy")
		subcmd.IntVar(&timestamp, "timestamp", 0, "Return all payment txs after the timestamp")
		if len(os.Args) < 2 {
			subcmd.PrintDefaults()
			os.Exit(1)
		}
		subcmd.Parse(os.Args[2:])
		if "" == currency || "" == addr {
			subcmd.PrintDefaults()
			os.Exit(1)
		}
		currency = strings.ToUpper(currency)
		query(currency, addr, timestamp)
	default:
		fmt.Printf("usage: payment-discovery [listen|query] <options>\n")
		os.Exit(1)
	}
}

func listen(currency string, addr string) {
	socket, _ := zmq.NewSocket(zmq.SUB)
	defer socket.Close()

	socket.Connect(addr)
	socket.SetSubscribe(currency)

	for {
		msg, err := socket.RecvMessageBytes(0)
		if err != nil {
			fmt.Printf("error: %s", err)
		}

		printJSONResult(msg[1])
	}
}

func query(currency string, addr string, timestamp int) {
	socket, _ := zmq.NewSocket(zmq.REQ)
	defer socket.Close()

	socket.Connect(addr)

	socket.SendMessage(currency, timestamp)
	msg, err := socket.RecvMessageBytes(0)
	if err != nil {
		fmt.Printf("error: %s", err)
	}

	status := string(msg[0])
	switch status {
	case "OK":
		printJSONResult(msg[1])
	case "ERROR":
		fmt.Printf("error: %s", string(msg[1]))
	}
}

func printJSONResult(data []byte) {
	buf := new(bytes.Buffer)
	json.Indent(buf, data, "", "  ")
	fmt.Println(buf)
}
