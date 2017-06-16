package main

type litecoinHandler struct {
	*bitcoinHandler
}

func newLitecoinHandler(name string, conf currencyConfig) *litecoinHandler {
	return &litecoinHandler{newBitcoinHandler(name, conf)}
}

func (l *litecoinHandler) Run() {
	go l.serveRequest()
	go l.listenBlockchain()
}
