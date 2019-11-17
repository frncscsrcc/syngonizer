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

func (f *feed) newError(err error) {
	f.errorChan <- err
}

func (f *feed) newFatal(err error) {
	f.fatalChan <- err
}

func (f *feed) newLog(message string) {
	f.logChan <- message
}
