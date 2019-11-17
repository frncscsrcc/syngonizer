package syngonizer

import (
	"strings"
)

func (wf *WatchFolder) write(path string) {
	remotePath := strings.Replace(path, wf.localRoot, "", 1)
	if wf.remoteRoot != "" {
		remotePath = wf.remoteRoot + remotePath
	}

	if isAFolder(path) {
		wf.writeFolder(remotePath)
		wf.existingFolders[path] = true
		return
	}

	wf.writeFile(path, remotePath)
	return
}

func (wf *WatchFolder) writeFolder(path string) {
	// Nothing to do if the folder already exists (eg: a file is written inside
	// a watched folder)
	if wf.existingFolders[path] {
		return
	}
	for _, app := range wf.apps {
		wf.kubeInfo.CreateFolder(app, path)
	}
	return
}

func (wf *WatchFolder) writeFile(localPath string, podPath string) {
	for _, app := range wf.apps {
		wf.kubeInfo.CopyFile(app, localPath, podPath)
	}
	return
}
