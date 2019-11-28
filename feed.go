package syngonizer

type feed struct {
	fatalChan chan error
	errorChan chan error
	logChan   chan string
}

var globalFeed *feed

func init() {
	globalFeed = new(feed)
	globalFeed.errorChan = make(chan error)
	globalFeed.fatalChan = make(chan error)
	globalFeed.logChan = make(chan string)
}
