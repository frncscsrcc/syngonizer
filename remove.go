package syngonizer

import (
	"strings"
)

func (wf *WatchFolder) remove(path string) {
	remotePath := strings.Replace(path, wf.localRoot, "", 1)
	if wf.remoteRoot != "" {
		remotePath = wf.remoteRoot + remotePath
	}

	if isAFolder(path) {
		wf.removeFolder(remotePath)
		delete(wf.existingFolders, path)
		return
	}

	wf.removeFile(remotePath)
	return
}

func (wf *WatchFolder) removeFolder(path string) {
	for _, app := range wf.apps {
		wf.kubeInfo.RemoveFolder(app, path)
	}
	return
}

func (wf *WatchFolder) removeFile(podPath string) {
	for _, app := range wf.apps {
		wf.kubeInfo.RemoveFile(app, podPath)
	}
	return
}
