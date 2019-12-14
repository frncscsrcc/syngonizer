package kubectl

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/frncscsrcc/syngonizer/config"
	"github.com/frncscsrcc/syngonizer/log"
)

// -----------------------------------------------------------------------------
// NOTE:
// This is a PoC WiP
// This section will be deleted, in order to use the official Kubernetes clien
// API!!!
// -----------------------------------------------------------------------------

// Connector ...
type Connector struct {
	config      config.Config
	l           sync.Mutex
	appToPods   map[string][]string
	namespace   string
	kubectlPath string
	// in order to avoid to create the remote folder all the time we update a file
	// format: {"POD123/folder/ABC" => true, "POD456/folder/ABC" => true, ...}
	folderCreatedOnPod map[string]bool
	log                log.Log
}

// NewConnector ...
func NewConnector(config config.Config, log log.Log) (*Connector, error) {
	c := new(Connector)

	// Avoid to send to many parallel command requestes via kubectls
	// this function is defined in command.go
	parallelServerRequestLimit := 10
	if config.Global.ParallelServerRequestLimit > 0 {
		parallelServerRequestLimit = config.Global.ParallelServerRequestLimit
	}
	log.SendLog(fmt.Sprintf("Setting a limit for parallel server requests: %d", parallelServerRequestLimit))
	initializeCommandLimiter(parallelServerRequestLimit)

	validationError := validate(config)
	if validationError != nil {
		return c, validationError
	}

	c.config = config
	c.appToPods = make(map[string][]string)
	c.namespace = config.Global.NameSpace
	c.kubectlPath = config.Global.KubectlPath
	// {pod123 => {folder1 => true, folder2 => true}, ...}
	c.folderCreatedOnPod = make(map[string]bool)
	c.log = log
	return c, nil
}

func validate(config config.Config) error {
	return nil
}

// UpdatePodListBackground ...
func (c *Connector) UpdatePodListBackground() {
	sleep := 60
	if c.config.Global.UpdatePodListInterval > 0 {
		sleep = c.config.Global.UpdatePodListInterval
	}

	ticker := time.NewTicker(time.Duration(sleep) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.UpdatePodList()
			}
		}
	}()
}

// UpdatePodList ...
func (c *Connector) UpdatePodList() error {
	c.l.Lock()
	defer c.l.Unlock()

	var podList PodList

	podListCommand := newCommand(c.kubectlPath, "-n", c.namespace, "get", "pods", "-o", "json")
	podListJSON, err := podListCommand.exec()
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(podListJSON), &podList)
	if err != nil {
		return err
	}

	// Map apps to pods
	newAppToPods := make(map[string][]string)
	for _, podItem := range podList.PodItems {
		app := podItem.Metadata.Labels.App
		name := podItem.Metadata.Name
		if _, exists := newAppToPods[app]; exists == false {
			newAppToPods[app] = make([]string, 0)
		}
		newAppToPods[app] = append(newAppToPods[app], name)
	}
	c.appToPods = newAppToPods

	// Reset the list of created folders
	c.folderCreatedOnPod = make(map[string]bool)

	if len(c.appToPods) == 0 {
		err := errors.New("no pods found in namespace " + c.namespace)
		c.log.SendFatal(err)
	}

	return err
}

// GetPodList ...
func (c *Connector) GetPodList(selector string) []string {
	c.l.Lock()
	defer c.l.Unlock()
	return c.appToPods[selector]
}

// CreateFolder ...
func (c *Connector) CreateFolder(app string, path string) {
	for _, podName := range c.GetPodList(app) {
		c.log.SendLog(fmt.Sprintf("%s %s: Creating folder %s\n", app, podName, path))
		createFolderCommand := newCommand(c.kubectlPath, "-n", c.namespace, "exec", podName, "--", "mkdir", "-p", path)
		execCommandsBackground(c.log, createFolderCommand)
	}
}

// WriteFile ...
func (c *Connector) WriteFile(app string, localPath string, podPath string) {
	for _, podName := range c.GetPodList(app) {
		// 1: Create the remote folder
		// The folder could not exists (folder creation and file creation ar
		// async processes. To avoid error, we force the folder creation inside the
		// container.
		remotePath, _ := filepath.Split(podPath)
		createFolderCommand := newCommand(c.kubectlPath,
			"-n", c.namespace, "exec", podName, "--", "mkdir", remotePath, "-p")
		// This command could fail if the folder already exists on the server or
		// if it is not possible create the folder. Just ignore. In case of errors
		// they will be reported in the next block
		createFolderCommand.ignoreErrors(true).beSilent(true)

		// 2: Write file
		writeFileCommand := newCommand(c.kubectlPath, "-n", c.namespace, "cp", localPath, podName+":"+podPath)

		// Try to create the folder only if we did not try previously on the same
		// pod.
		if c.folderCreatedOnPod[podName+remotePath] == false {
			c.folderCreatedOnPod[podName+remotePath] = true
			c.log.SendLog(fmt.Sprintf("%s %s: Writing file %s (+ create folder, if required)\n", app, podName, podPath))
			execCommandsBackground(c.log, createFolderCommand, writeFileCommand)
		} else {
			c.log.SendLog(fmt.Sprintf("%s %s: Writing file %s\n", app, podName, podPath))
			execCommandsBackground(c.log, writeFileCommand)
		}
	}
}

// RemoveFolder ...
func (c *Connector) RemoveFolder(app string, path string) {
	for _, podName := range c.GetPodList(app) {
		c.log.SendLog(fmt.Sprintf("%s %s: Removing folder %s\n", app, podName, path))
		removeFolderCommand := newCommand(c.kubectlPath, "-n", c.namespace, "exec", podName, "--", "rmdir", path)
		execCommandsBackground(c.log, removeFolderCommand)
	}
}

// RemoveFile ...
func (c *Connector) RemoveFile(app string, path string) {
	for _, podName := range c.GetPodList(app) {
		c.log.SendLog(fmt.Sprintf("%s %s: Removing file %s\n", app, podName, path))
		removeFileCommand := newCommand(c.kubectlPath, "-n", c.namespace, "exec", podName, "--", "rm", path)
		execCommandsBackground(c.log, removeFileCommand)
	}
}

func newError(app string, podName string, err error) error {
	return errors.New(fmt.Sprintf("%s %s: ERROR: %s", app, podName, err.Error()))
}
