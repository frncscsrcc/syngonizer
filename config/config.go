package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"
)

// Config ...
type Config struct {
	Global  GlobalConfig   `json:"global"`
	Folders []FolderConfig `json:"folders"`
}

// GlobalConfig ...
type GlobalConfig struct {
	EventListenInterval   float64 `json:"event-listen-iterval"`
	NameSpace             string  `json:"namespace"`
	KubectlPath           string  `json:"kubectl-path"`
	UpdatePodListInterval int     `json:"update-pod-list-interval"`
	AllowProduction       bool    `json:"allow-production"`
	DieIfError            bool    `json:"die-if-error"`
}

// FolderConfig ...
type FolderConfig struct {
	LocalRoot  string   `json:"local-root"`
	RemoteRoot string   `json:"remote-root"`
	Apps       []string `json:"apps"`
}

// LoadConfig ...
func LoadConfig(configPath string) (Config, error) {
	var config Config
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal([]byte(file), &config)
	if err != nil {
		return config, err
	}

	// ----------------
	// Validations
	// ----------------
	if config.Global.NameSpace == "production" && config.Global.AllowProduction != true {
		return config, errors.New("can not be used in production namespace")
	}
	if config.Global.UpdatePodListInterval > 0 && config.Global.UpdatePodListInterval < 5 {
		return config, errors.New("update-pod-list-interval min value is 5")
	}
	for _, folderConfig := range config.Folders {
		// local-root is required
		if folderConfig.LocalRoot == "" {
			return config, errors.New("missing local-root in config")
		}
		// local-root must be an absolute path
		if strings.HasPrefix(folderConfig.LocalRoot, "/") == false {
			return config, errors.New("local-root must be an absolute path")
		}
		// local-root must exists and be a folder
		if !isAFolder(folderConfig.LocalRoot) {
			return config, errors.New("local-root " + folderConfig.LocalRoot + " is not a folder")
		}
		// remote-root, if present, must be an absolute path
		if folderConfig.RemoteRoot != "" && strings.HasPrefix(folderConfig.RemoteRoot, "/") == false {
			return config, errors.New("remote-root must be an absolute path")
		}
		// app selector is required
		if len(folderConfig.Apps) == 0 {
			return config, errors.New("missing apps selector")
		}
	}

	return config, nil
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
