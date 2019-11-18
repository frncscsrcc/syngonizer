package syngonizer

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
)

// Config ...
type Config struct {
	Global  GlobalConfig   `json:"global"`
	Folders []FolderConfig `json:"folders"`
}

// GlobalConfig ...
type GlobalConfig struct {
	RefreshRate     float64 `json:"refresh-rate"`
	NameSpace       string  `json:"namespace"`
	KubectlPath     string  `json:"kubectl-path"`
	RefreshPodList  int     `json:"refresh-pod-list"`
	AllowProduction bool    `json:"allow-production"`
	DieIfError      bool    `json:"die-if-error"`
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
	for _, folderConfig := range config.Folders {
		// local-root is required
		if folderConfig.LocalRoot == "" {
			return config, errors.New("missing local-root in config")
		}
		// local-root must be an absolute path
		if strings.HasPrefix(folderConfig.LocalRoot, "/") == false {
			return config, errors.New("local-root must be an absolute path")
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
