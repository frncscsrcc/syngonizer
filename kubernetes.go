package syngonizer

import (
	"encoding/json"
	"log"
	"path/filepath"
	"sync"
	"time"
)

// -----------------------------------------------------------------------------
// NOTE:
// This is a PoC WiP
// This section will be deleted, in order to use the official Kubernetes clien
// API!!!
// -----------------------------------------------------------------------------

// PodMetadata ...
type PodMetadata struct {
	Name      string    `json:"name"`
	Labels    PodLabels `json:"labels"`
	Namespace string    `json:"namespace"`
}

// PodLabels ...
type PodLabels struct {
	App string `json:"app"`
}

// PodItem ...
type PodItem struct {
	Metadata PodMetadata `json:"metadata"`
}

// PodList ...
type PodList struct {
	PodItems []PodItem `json:"items"`
}

// KubeInfo ...
type KubeInfo struct {
	l           sync.Mutex
	appToPods   map[string][]string
	namespace   string
	kubectlPath string
	// in order to avoid to create the remote folder all the time we update a file
	// format: {"POD123/folder/ABC" => true, "POD456/folder/ABC" => true, ...}
	folderCreatedOnPod map[string]bool
}

// NewKubeInfo ...
func NewKubeInfo(namespace string, kubectlPath string) *KubeInfo {
	ki := new(KubeInfo)
	ki.appToPods = make(map[string][]string)
	ki.namespace = namespace
	ki.kubectlPath = kubectlPath
	// {pod123 => {folder1 => true, folder2 => true}, ...}
	ki.folderCreatedOnPod = make(map[string]bool)
	return ki
}

// UpdatePodListBackground ...
func (ki *KubeInfo) UpdatePodListBackground(sleep int) {
	ticker := time.NewTicker(time.Duration(sleep) * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				ki.UpdatePodList()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// UpdatePodList ...
func (ki *KubeInfo) UpdatePodList() error {
	ki.l.Lock()
	defer ki.l.Unlock()

	var podList PodList

	podListCommand := newCommand(ki.kubectlPath, "-n", ki.namespace, "get", "pods", "-o", "json")
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
	ki.appToPods = newAppToPods

	// Reset the list of created folders
	ki.folderCreatedOnPod = make(map[string]bool)

	if len(ki.appToPods) == 0 {
		log.Fatal("no pods found in namespace" + ki.namespace + "\n")
	}

	return err
}

// CreateFolder ...
func (ki *KubeInfo) CreateFolder(app string, path string) {
	ki.l.Lock()
	defer ki.l.Unlock()

	pods := ki.appToPods[app]
	for _, podName := range pods {
		globalFeed.newLog("Creating folder" + path + " in pod " + podName)

		createFolderCommand := newCommand(ki.kubectlPath, "-n", ki.namespace, "exec", podName, "--", "mkdir", path)
		backgroundExecCommands(createFolderCommand)
	}
}

// WriteFile ...
func (ki *KubeInfo) WriteFile(app string, localPath string, podPath string) {
	ki.l.Lock()
	defer ki.l.Unlock()

	pods := ki.appToPods[app]
	for _, podName := range pods {

		// 1: Create the remote folder
		// The folder could not exists (folder creation and file creation ar
		// async processes. To avoid error, we force the folder creation inside the
		// container.
		remotePath, _ := filepath.Split(podPath)
		createFolderCommand := newCommand(ki.kubectlPath,
			"-n", ki.namespace, "exec", podName, "--", "mkdir", remotePath)
		// This command could fail if the folder already exists on the server or
		// if it is not possible create the folder. Just ignore. In case of errors
		// they will be reported in the next block
		createFolderCommand.ignoreErrors(true).beSilent(true)

		// 2: Write file
		writeFileCommand := newCommand(ki.kubectlPath, "-n", ki.namespace, "cp", localPath, podName+":"+podPath)

		// Try to create the folder only if we did not try previously on the same
		// pod.
		if ki.folderCreatedOnPod[podName+remotePath] == false {
			ki.folderCreatedOnPod[podName+remotePath] = true
			globalFeed.newLog("Creating folder " + remotePath + " in pod " + podName)
			globalFeed.newLog("Writing file " + podPath + " in pod " + podName)
			backgroundExecCommands(createFolderCommand, writeFileCommand)
		} else {
			globalFeed.newLog("Writing file " + podPath + " in pod " + podName)
			backgroundExecCommands(writeFileCommand)
		}
	}
}

// RemoveFolder ...
func (ki *KubeInfo) RemoveFolder(app string, path string) {
	ki.l.Lock()
	defer ki.l.Unlock()

	pods := ki.appToPods[app]
	for _, podName := range pods {
		globalFeed.newLog("Removing folder" + path + " in pod " + podName)

		removeFolderCommand := newCommand(ki.kubectlPath, "-n", ki.namespace, "exec", podName, "--", "rmdir", path)
		backgroundExecCommands(removeFolderCommand)
	}
}

// RemoveFile ...
func (ki *KubeInfo) RemoveFile(app string, path string) {
	ki.l.Lock()
	defer ki.l.Unlock()

	pods := ki.appToPods[app]
	for _, podName := range pods {
		globalFeed.newLog("Removing file " + path + " in pod " + podName)

		removeFileCommand := newCommand(ki.kubectlPath, "-n", ki.namespace, "exec", podName, "--", "rm", path)
		backgroundExecCommands(removeFileCommand)
	}
}
