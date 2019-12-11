package syngonizer

import (
	"errors"
	"log"
	"time"

	"github.com/frncscsrcc/syngonizer/config"
	"github.com/frncscsrcc/syngonizer/connector"
	"github.com/radovskyb/watcher"
)

// WatchFolder ...
type WatchFolder struct {
	watcher             *watcher.Watcher
	eventQueue          chan watcher.Event
	localRoot           string
	remoteRoot          string
	apps                []string
	existingFolders     map[string]bool
	eventListenInterval float64
	connector           connector.Connector
}

// LoadConfig ...
func LoadConfig(path string) (config.Config, error) {
	return config.LoadConfig(path)
}

func initEventQueue(numWorkers int) {
	startWorker := func() {
		for {
			select {
			case event := <-eventQueue:
				switch event.Op {
				case watcher.Write:
					wf.write(event.Path)
				case watcher.Create:
					wf.write(event.Path)
				case watcher.Remove:
					wf.remove(event.Path)
				case watcher.Rename:
					// wf.move(event.Path)
				}
			}
		}
	}

	for i := 0; i < numWorkers; i++ {
		go startWorker()
	}
}

// Watch ...
func Watch(config config.Config) error {
	c, err := connector.NewConnector(config, globalFeed.logChan, globalFeed.errorChan)
	if err != nil {
		return errors.New("can not create a connector")
	}

	// Load app to pods (async the first time, so it can return if can not connect)
	errUpdate := c.UpdatePodList()
	if errUpdate != nil {
		log.Printf("%s\n", errUpdate)
		return errors.New("can not fetch pod list for namespace " + config.Global.NameSpace)
	}
	log.Printf("Fetched app to pod list for namespace %s\n", config.Global.NameSpace)

	c.UpdatePodListBackground()

	watchers := make([]*WatchFolder, 0)
	for _, folderConfig := range config.Folders {
		w, err := addWatchFolder(config.Global.EventListenInterval, config.Global.WorkersLimit, folderConfig, c)
		if err != nil {
			return err
		}
		watchers = append(watchers, w)
	}

	for _, w := range watchers {
		// LISTEN for events!!
		go w.listen()

		log.Printf("Watching %s\n", w)
	}

	// Handle errors
loop:
	for {
		select {
		case errMessage := <-globalFeed.errorChan:
			log.Printf("ERROR: %s\n", errMessage)
			if config.Global.DieIfError {
				log.Fatal("Die because die-if-error is set true")
			}
		case logMessage := <-globalFeed.logChan:
			log.Printf("%s\n", logMessage)
		case fatalMessage := <-globalFeed.fatalChan:
			log.Printf("ERROR: %s\n", fatalMessage)
			break loop
		}
	}

	return nil
}

func addWatchFolder(eventListenInterval float64, folderConfig config.FolderConfig, workerLimit int, c connector.Connector) (*WatchFolder.Connector) (*WorkersLimit, , error) {
	wf := new(WatchFolder)
	root := folderConfig.LocalRoot
	if isAFolder(root) == false {
		return wf, errors.New(root + " is not a folder")
	}

	wf.localRoot = root
	wf.remoteRoot = folderConfig.RemoteRoot
	wf.eventListenInterval = eventListenInterval
	wf.apps = folderConfig.Apps
	wf.connector = c
	wf.eventQueue = make(chan watcher.Event)
	wf.existingFolders = make(map[string]bool)

	wf.watcher = watcher.New()
	if err := wf.watcher.AddRecursive(root); err != nil {
		return wf, err
	}

	// Mark the existing folders
	for path := range wf.watcher.WatchedFiles() {
		if isAFolder(path) {
			wf.existingFolders[path] = true
		}
	}

	// Initialize the workers (event consumers)
	for i := 0; i < workerLimit; i++ {
		go startWorker(wf)
	}

	return wf, nil
}

func (wf *WatchFolder) String() string {
	return wf.localRoot
}

// Watch ..
func (wf *WatchFolder) listen() {
	// Event handler
	go func() {
		for {
			select {
			// Forward event to consumer queue
			case event := <-wf.watcher.Event:
				wf.eventQueue <-event
			case err := <-wf.watcher.Error:
				globalFeed.errorChan <- err
			case <-wf.watcher.Closed:
				globalFeed.logChan <- "closing chanel"
				return
			}
		}
	}()
	// Set refresh rate
	eventListenInterval := _eventListenInterval
	if wf.eventListenInterval > 0 {
		eventListenInterval = wf.eventListenInterval
	}
	// Sec to ms
	eventListenInterval = 1000 * eventListenInterval

	// Start listening events
	if err := wf.watcher.Start(time.Millisecond * time.Duration(eventListenInterval)); err != nil {
		globalFeed.errorChan <- err
	}
}

func startWorker(wf *WatchFolder) {
	for {
		select {
		case event := <-wf.eventQueue:
			switch event.Op {
			case watcher.Write:
				wf.write(event.Path)
			case watcher.Create:
				wf.write(event.Path)
			case watcher.Remove:
				wf.remove(event.Path)
			case watcher.Rename:
				// wf.move(event.Path)
			}
		}
	}
}
