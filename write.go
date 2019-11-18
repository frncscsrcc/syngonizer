package syngonizer

import (
	"os"
	"strings"
)

func (wf *WatchFolder) write(path string) {
	remotePath := strings.Replace(path, wf.localRoot, "", 1)
	if wf.remoteRoot != "" {
		remotePath = wf.remoteRoot + remotePath
	}

	if isAFolder(path) {
		// Nothing to do if the folder already exists (eg: a file is written inside
		// a watched folder)
		if wf.existingFolders[path] {
			return
		}
		wf.writeFolder(remotePath)
		wf.existingFolders[path] = true
		return
	}

	wf.writeFile(path, remotePath)
	return
}

func (wf *WatchFolder) writeFolder(path string) {
	for _, app := range wf.apps {
		wf.kubeInfo.CreateFolder(app, path)
	}
	return
}

func (wf *WatchFolder) writeFile(localPath string, podPath string) {
	for _, app := range wf.apps {
		wf.kubeInfo.WriteFile(app, localPath, podPath)
	}
	return
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
