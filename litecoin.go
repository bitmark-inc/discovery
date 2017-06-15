package main

type litecoinHandler struct {
	*bitcoinHandler
}

func newLitecoinHandler(name string, conf currencyConfig) *litecoinHandler {
	return &litecoinHandler{newBitcoinHandler(name, conf)}
}

func (l *litecoinHandler) Run() {
	done := make(chan struct{})
	go l.rescanRecentBlocks(done)
	go l.serveRequest(done)

	go l.listenBlockchain()
}
