package syngonizer

import (
	"os"
	"strings"
)

func (wf *WatchFolder) write(path string) ([]string, []error) {
	remotePath := strings.Replace(path, wf.localRoot, "", 1)
	if wf.remoteRoot != "" {
		remotePath = wf.remoteRoot + remotePath
	}

	if isAFolder(path) {
		// Nothing to do if the folder already exists (eg: a file is written inside
		// a watched folder)
		if wf.existingFolders[path] {
			return []string{}, nil
		}
		logs, errs := wf.writeFolder(remotePath)
		if len(errs) == 0 {
			wf.existingFolders[path] = true
		}
		return logs, errs
	}

	logs, errs := wf.writeFile(path, remotePath)
	return logs, errs
}

func (wf *WatchFolder) writeFolder(path string) ([]string, []error) {
	logSlice := make([]string, 0)
	errorSlice := make([]error, 0)
	for _, app := range wf.apps {
		logs, errs := wf.connector.CreateFolder(app, path)
		for _, m := range(logs){
			logSlice = append(logSlice, m)
		}
		for _, m := range(errs){
			errorSlice = append(errorSlice, m)
		}		
	}
	return logSlice, errorSlice
}

func (wf *WatchFolder) writeFile(localPath string, podPath string) ([]string, []error) {
	logSlice := make([]string, 0)
	errorSlice := make([]error, 0)
	for _, app := range wf.apps {
		logs, errs := wf.connector.WriteFile(app, localPath, podPath)
		for _, m := range(logs){
			logSlice = append(logSlice, m)
		}
		for _, m := range(errs){
			errorSlice = append(errorSlice, m)
		}		
	}
	return logSlice, errorSlice
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
