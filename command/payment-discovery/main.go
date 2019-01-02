// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	zmq "github.com/pebbe/zmq4"
)

var (
	currency  string
	addr      string
	timestamp int
)

func main() {
	subcmd := flag.NewFlagSet("subcmd", flag.ExitOnError)
	subcmd.StringVar(&currency, "currency", "btc", "Specify the currency (possible values: btc, ltc)")
	subcmd.StringVar(&addr, "addr", "", "Specify the zeromq socket address of discovery proxy")
	subcmd.IntVar(&timestamp, "timestamp", 0, "Return all payment txs after the timestamp")
	subcmd.Parse(os.Args[2:])

	currency = strings.ToUpper(currency)

	switch os.Args[1] {
	case "listen":
		listen(currency, addr)
	case "query":
		query(addr, timestamp)
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func listen(currency, addr string) {
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

func query(addr string, timestamp int) {
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
