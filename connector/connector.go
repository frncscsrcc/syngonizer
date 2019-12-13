package connector

import (
	"github.com/frncscsrcc/syngonizer/config"
	"github.com/frncscsrcc/syngonizer/connector/kubectl"
)

// Connector ...
type Connector interface {
	UpdatePodListBackground()
	UpdatePodList() error

	CreateFolder(string, string) ([]string, []error)
	WriteFile(string, string, string) ([]string, []error)
	RemoveFolder(string, string) ([]string, []error)
	RemoveFile(string, string) ([]string, []error)
}

// NewConnector ...
// At the moment we handle only Kubectl Connector
func NewConnector(config config.Config) (Connector, error) {
	return kubectl.NewConnector(config)
}
