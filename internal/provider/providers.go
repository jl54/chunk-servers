package provider

import (
	"errors"
	"fmt"
)

type ServerProvider interface {
	CreateServer() (string, error)
	ReadServer()
	ListServers()
	UpdateServer()
	DeleteServer(vmid int)
	ConfigureFromEnvironment() error
}

func New(provider string) (ServerProvider, error) {
	switch provider {
	case "proxmox":
		return &ProxmoxProvider{}, nil
	default:
		return nil, errors.New(fmt.Sprintf("unknown provider: %s", provider))
	}
}
