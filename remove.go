package syngonizer

import (
	"strings"
)

func (wf *WatchFolder) remove(path string) {
	relativePath := strings.Replace(path, wf.rootFolder, "", 1)

	if isAFolder(path) {
		wf.removeFolder(relativePath)
		delete(wf.existingFolders, path)
		return
	}

	wf.removeFile(relativePath)
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
