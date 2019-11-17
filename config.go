package syngonizer

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
)

// Config ...
type Config struct {
	Global  GlobalConfig   `json:"global"`
	Folders []FolderConfig `json:"folders"`
}

// GlobalConfig ...
type GlobalConfig struct {
	RefreshRate     int    `json:"refresh-rate"`
	NameSpace       string `json:"namespace"`
	KubectlPath     string `json:"kubectl-path"`
	RefreshPodList  int    `json:"refresh-pod-list"`
	AllowProduction bool   `json:"allow-production"`
	DieIfError      bool   `json:"die-if-error"`
}

// FolderConfig ...
type FolderConfig struct {
	Root string   `json:"root"`
	Apps []string `json:"apps"`
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
		log.Fatal(err)
	}

	if config.Global.NameSpace == "production" && config.Global.AllowProduction != true {
		log.Fatal(errors.New("can not be used in production namespace"))
	}
	return config, nil
}
