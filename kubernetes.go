package syngonizer

import (
	"encoding/json"
	"log"
	"os/exec"
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
}

// NewKubeInfo ...
func NewKubeInfo(namespace string, kubectlPath string) *KubeInfo {
	ki := new(KubeInfo)
	ki.appToPods = make(map[string][]string)
	ki.namespace = namespace
	ki.kubectlPath = kubectlPath
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
	podListJSON, err := execCommand(ki.kubectlPath, "-n", ki.namespace, "get", "pods", "-o", "json")
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(podListJSON), &podList)
	if err != nil {
		return err
	}

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
		backgroundExecCommand(ki.kubectlPath, "-n", ki.namespace, "exec", podName, "--", "touch", path)
	}
}

// CopyFile ...
func (ki *KubeInfo) CopyFile(app string, localPath string, podPath string) {
	ki.l.Lock()
	defer ki.l.Unlock()

	pods := ki.appToPods[app]
	for _, podName := range pods {
		globalFeed.newLog("Copying file " + localPath + " to " + podPath + " in pod " + podName)
		backgroundExecCommand(ki.kubectlPath, "-n", ki.namespace, "cp", localPath, podName+":"+podPath)
	}
}

// RemoveFolder ...
func (ki *KubeInfo) RemoveFolder(app string, path string) {
	ki.l.Lock()
	defer ki.l.Unlock()

	pods := ki.appToPods[app]
	for _, podName := range pods {
		globalFeed.newLog("Removing folder" + path + " in pod " + podName)
		backgroundExecCommand(ki.kubectlPath, "-n", ki.namespace, "exec", podName, "--", "rmdir", path)
	}
}

// RemoveFile ...
func (ki *KubeInfo) RemoveFile(app string, path string) {
	ki.l.Lock()
	defer ki.l.Unlock()

	pods := ki.appToPods[app]
	for _, podName := range pods {
		globalFeed.newLog("Removing file " + path + " in pod " + podName)
		backgroundExecCommand(ki.kubectlPath, "-n", ki.namespace, "exec", podName, "--", "rm", path)
	}
}

func backgroundExecCommand(cmd string, args ...string) {
	go execCommand(cmd, args...)
}

func execCommand(cmd string, args ...string) (string, error) {
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		globalFeed.newError(err)
	}
	return string(out), err
}
