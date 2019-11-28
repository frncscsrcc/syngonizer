package connector

import (
	"github.com/frncscsrcc/syngonizer/config"
	"github.com/frncscsrcc/syngonizer/connector/kubectl"
)

// Connector ...
type Connector interface {
	UpdatePodListBackground()
	UpdatePodList() error

	CreateFolder(string, string)
	WriteFile(string, string, string)
	RemoveFolder(string, string)
	RemoveFile(string, string)
}

// NewConnector ...
// At the moment we handle only Kubectl Connector
func NewConnector(config config.Config, logChan chan string, errChan chan error) (Connector, error) {
	return kubectl.NewConnector(config, logChan, errChan)
}
