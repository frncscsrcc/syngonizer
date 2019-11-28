package kubectl

import (
	"encoding/json"
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
	logChan            chan string
	errChan            chan error
}

// NewConnector ...
func NewConnector(config config.Config, logChan chan string, errChan chan error) (*Connector, error) {
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
	c.logChan = logChan
	c.errChan = errChan
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
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				c.UpdatePodList()
			case <-quit:
				ticker.Stop()
				return
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
func (c *Connector) CreateFolder(app string, path string) {
	for _, podName := range c.GetPodList(app) {
		c.logChan <- "Creating folder" + path + " in pod " + podName

		createFolderCommand := newCommand(c.kubectlPath, "-n", c.namespace, "exec", podName, "--", "mkdir", path)
		backgroundExecCommands(c.errChan, createFolderCommand)
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
			c.logChan <- "Creating folder " + remotePath + " in pod " + podName
			c.logChan <- "Writing file " + podPath + " in pod " + podName
			backgroundExecCommands(c.errChan, createFolderCommand, writeFileCommand)
		} else {
			c.logChan <- "Writing file " + podPath + " in pod " + podName
			backgroundExecCommands(c.errChan, writeFileCommand)
		}
	}
}

// RemoveFolder ...
func (c *Connector) RemoveFolder(app string, path string) {
	for _, podName := range c.GetPodList(app) {
		c.logChan <- "Removing folder" + path + " in pod " + podName

		removeFolderCommand := newCommand(c.kubectlPath, "-n", c.namespace, "exec", podName, "--", "rmdir", path)
		backgroundExecCommands(c.errChan, removeFolderCommand)
	}
}

// RemoveFile ...
func (c *Connector) RemoveFile(app string, path string) {
	for _, podName := range c.GetPodList(app) {
		c.logChan <- "Removing file " + path + " in pod " + podName

		removeFileCommand := newCommand(c.kubectlPath, "-n", c.namespace, "exec", podName, "--", "rm", path)
		backgroundExecCommands(c.errChan, removeFileCommand)
	}
}
