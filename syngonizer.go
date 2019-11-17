package syngonizer

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/radovskyb/watcher"
)

// WatchFolder ...
type WatchFolder struct {
	watcher         *watcher.Watcher
	rootFolder      string
	apps            []string
	existingFolders map[string]bool
	refreshRate     int
	kubeInfo        *KubeInfo
}

// Watch ...
func Watch(config Config) error {
	ki := NewKubeInfo(config.Global.NameSpace, config.Global.KubectlPath)
	// Load app to pods (async the first time, so it can return if can not connect)
	err := ki.UpdatePodList()
	if err != nil {
		log.Printf("%s\n", err)
		return errors.New("can not fetch pod list for namespace " + config.Global.NameSpace)
	}
	log.Printf("Fetched app to pod list for namespace %s\n", config.Global.NameSpace)

	// Request a refresh on pod list based on time interval
	refreshPodList := 60
	if config.Global.RefreshPodList > 0 {
		refreshPodList = config.Global.RefreshPodList
	}
	ki.UpdatePodListBackground(refreshPodList)

	watchers := make([]*WatchFolder, 0)
	for _, folderConfig := range config.Folders {
		w, err := addWatchFolder(config.Global, folderConfig, ki)
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

func addWatchFolder(globalConfig GlobalConfig, folderConfig FolderConfig, ki *KubeInfo) (*WatchFolder, error) {
	wf := new(WatchFolder)
	root := folderConfig.Root
	if isAFolder(root) == false {
		return wf, errors.New(root + " is not a folder")
	}

	wf.rootFolder = root
	wf.refreshRate = globalConfig.RefreshRate
	wf.apps = folderConfig.Apps
	wf.kubeInfo = ki

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

	return wf, nil
}

func (wf *WatchFolder) String() string {
	return wf.rootFolder
}

// Watch ..
func (wf *WatchFolder) listen() {
	// Event handler
	go func() {
		for {
			select {
			case event := <-wf.watcher.Event:
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
			case err := <-wf.watcher.Error:
				globalFeed.newError(err)
			case <-wf.watcher.Closed:
				globalFeed.newLog("closing chanel")
				return
			}
		}
	}()

	// Set refresh rate
	refreshRate := _refreshRate
	if wf.refreshRate > 0 {
		refreshRate = wf.refreshRate
	}

	// Start listening events
	if err := wf.watcher.Start(time.Millisecond * time.Duration(refreshRate)); err != nil {
		globalFeed.newError(err)
	}
}

func isAFolder(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	fi, err1 := file.Stat()
	switch {
	case err1 != nil:
		return false
	case fi.IsDir():
		return true
	default:
		return false
	}
}
