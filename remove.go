package syngonizer

import (
	"strings"
)

func (wf *WatchFolder) remove(path string) ([]string, []error) {
	remotePath := strings.Replace(path, wf.localRoot, "", 1)
	if wf.remoteRoot != "" {
		remotePath = wf.remoteRoot + remotePath
	}

	// Is a folder (can not check in the FS, because the folder is already deleted)
	if wf.existingFolders[path] {
		logs, errs := wf.removeFolder(remotePath)
		if len(errs) == 0 {
			delete(wf.existingFolders, path)
		}
		return logs, errs
	}

	logs, errs := wf.removeFile(remotePath)
	return logs, errs
}

func (wf *WatchFolder) removeFolder(path string) ([]string, []error) {
	logSlice := make([]string, 0)
	errorSlice := make([]error, 0)
	for _, app := range wf.apps {
		logs, errs := wf.connector.RemoveFolder(app, path)
		for _, m := range(logs){
			logSlice = append(logSlice, m)
		}
		for _, m := range(errs){
			errorSlice = append(errorSlice, m)
		}	
	}
	return logSlice, errorSlice
}

func (wf *WatchFolder) removeFile(podPath string) ([]string, []error) {
	logSlice := make([]string, 0)
	errorSlice := make([]error, 0)
	for _, app := range wf.apps {
		logs, errs := wf.connector.RemoveFile(app, podPath)
		for _, m := range(logs){
			logSlice = append(logSlice, m)
		}
		for _, m := range(errs){
			errorSlice = append(errorSlice, m)
		}	
	}
	return logSlice, errorSlice
}
