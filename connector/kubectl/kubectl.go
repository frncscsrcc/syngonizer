package kubectl

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/frncscsrcc/syngonizer/config"
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
}

// NewConnector ...
func NewConnector(config config.Config) (*Connector, error) {
	c := new(Connector)

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
		log.Fatal("no pods found in namespace" + c.namespace + "\n")
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
func (c *Connector) CreateFolder(app string, path string) ([]string, []error) {
	logSlice := make([]string, 0)
	errorSlice := make([]error, 0)
	for _, podName := range c.GetPodList(app) {
		logSlice = append(logSlice, fmt.Sprintf("%s %s: Creating folder %s\n", app, podName, path))
		createFolderCommand := newCommand(c.kubectlPath, "-n", c.namespace, "exec", podName, "--", "mkdir", "-p", path)
		_, err := execCommands(createFolderCommand)
		if err != nil {
			errorSlice = append(errorSlice, newError(app, podName, err))
		}
	}
	return logSlice, errorSlice
}

// WriteFile ...
func (c *Connector) WriteFile(app string, localPath string, podPath string) ([]string, []error) {
	logSlice := make([]string, 0)
	errorSlice := make([]error, 0)

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
			logSlice = append(logSlice, fmt.Sprintf("%s %s: Creating folder %s\n", app, podName, remotePath))
			_, err := execCommands(createFolderCommand, writeFileCommand)
			if err != nil {
				errorSlice = append(errorSlice, newError(app, podName, err))
			}
		}

		logSlice = append(logSlice, fmt.Sprintf("%s %s: Writing file %s\n", app, podName, podPath))
		_, err := execCommands(writeFileCommand)
		if err != nil {
			errorSlice = append(errorSlice, newError(app, podName, err))
		}
	}
	return logSlice, errorSlice
}

// RemoveFolder ...
func (c *Connector) RemoveFolder(app string, path string) ([]string, []error) {
	logSlice := make([]string, 0)
	errorSlice := make([]error, 0)

	for _, podName := range c.GetPodList(app) {
		logSlice = append(logSlice, fmt.Sprintf("%s %s: Removing folder %s\n", app, podName, path))
		removeFolderCommand := newCommand(c.kubectlPath, "-n", c.namespace, "exec", podName, "--", "rmdir", path)
		_, err := execCommands(removeFolderCommand)
		if err != nil {
			errorSlice = append(errorSlice, newError(app, podName, err))
		}
	}
	return logSlice, errorSlice
}

// RemoveFile ...
func (c *Connector) RemoveFile(app string, path string) ([]string, []error) {
	logSlice := make([]string, 0)
	errorSlice := make([]error, 0)

	for _, podName := range c.GetPodList(app) {
		logSlice = append(logSlice, fmt.Sprintf("%s %s: Removing file %s\n", app, podName, path))
		removeFileCommand := newCommand(c.kubectlPath, "-n", c.namespace, "exec", podName, "--", "rm", path)
		_, err := execCommands(removeFileCommand)
		if err != nil {
			errorSlice = append(errorSlice, newError(app, podName, err))
		}
	}
	return logSlice, errorSlice
}

func newError(app string, podName string, err error) error {
	return errors.New(fmt.Sprintf("%s %s: ERROR: %s", app, podName, err.Error()))
}