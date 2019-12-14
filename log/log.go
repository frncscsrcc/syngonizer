package log

import (
	"log"
)

// Level ...
type Level int

const (
	// DEBUG ...
	DEBUG Level = iota
	// LOG ...
	LOG
	// ERROR ...
	ERROR
	// FATAL ...
	FATAL
)

// Log ...
type Log struct {
	logLevel      Level
	debug         chan string
	log           chan string
	notFatalError chan error
	fatal         chan error
	done          chan bool
}

// NewLog ...
func NewLog(logLevel Level) Log {
	return Log{
		logLevel:      logLevel,
		debug:         make(chan string),
		log:           make(chan string),
		notFatalError: make(chan error),
		fatal:         make(chan error),
		done:          make(chan bool),
	}
}

// SendDebug ..
func (l Log) SendDebug(message string) {
	go func(message string) {
		if DEBUG >= l.logLevel {
			l.debug <- message
		}
	}(message)
}

// SendLog ..
func (l Log) SendLog(message string) {
	go func(message string) {
		if LOG >= l.logLevel {
			l.log <- message
		}
	}(message)
}

// SendError ..
func (l Log) SendError(err error) {
	go func(err error) {
		if ERROR >= l.logLevel {
			l.notFatalError <- err
		}
	}(err)
}

// SendFatal ..
func (l Log) SendFatal(err error) {
	go func(err error) {
		l.notFatalError <- err
	}(err)
}

// PrintLog ..
func (l Log) PrintLog() {
	for {
		select {
		case message := <-l.debug:
			log.Printf("DEBUG: %s", message)
		case message := <-l.log:
			log.Printf("LOG:   %s", message)
		case err := <-l.notFatalError:
			log.Printf("ERROR: %s", err.Error())
		case err := <-l.fatal:
			log.Printf("FATAL: %s", err.Error())
			log.Panic(err)
		case <-l.done:
			return
		}
	}
}
