package provider

import (
	"errors"
	"fmt"
)

type ServerProvider interface {
	CreateServer(name string, memory, cores int) error
	ReadServer(id int) (VmGetResponse, error)
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
