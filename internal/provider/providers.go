package provider

import (
	"errors"
	"fmt"
)

type ServerProvider interface {
	CreateServer() (string, error)
	ReadServer()
	UpdateServer()
	DeleteServer()
}

type ProxmoxProvider struct {
	Host        string
	Port        int
	Username    string
	TokenId     string
	TokenSecret string
}

type ProxmoxServerOptions struct {
	Required struct {
		Node string
		Vmid int
	}
	Optional struct {
		Acpi        bool
		Affinity    string
		Agent       string
		AmdSev      string
		Arch        string
		Archive     string
		Args        string
		Audio0      string
		Autostart   bool
		Balloon     int
		Bios        string
		Boot        string
		Bootdisk    string
		Bwlimit     int
		Cdrom       string
		Cicustom    string
		Cipassword  string
		Citype      string
		Ciupgrade   bool
		Ciuser      string
		Cores       int
		Cpu         string
		Cpulimit    int
		Cpuunits    int
		Description string
		Efidisk0    string
		Force       bool
		Freeze      bool
		Hookscript  string
		// Hostpci[n] string
		Hotplug   string
		Hugepages string
		// Ide[n] string
		ImportWorkingStorage string
		// Ipconfig[n] string
		Ivshmem          string
		Keephugepages    bool
		Keyboard         string
		Kvm              bool
		LiveRestore      bool
		Localtime        bool
		Lock             string
		Machine          string
		Memory           string
		Migrate_downtime int
		Migrate_speed    int
		Name             string
		Nameserver       string
		// Net[n] string
		Numa bool
		// Numa[n] string
		Onboot bool
		Ostype string
		// Parallel[n] string
		Pool       string
		Protection bool
		Reboot     bool
		Rng0       string
		// Sata[n] string
		// Scsi[n] string
		Scsihw       string
		Searchdomain string
		// Serial[n] string
		Shares             int
		Smbios1            string
		Smp                int
		Sockets            int
		Spice_enhancements string
		Sshkeys            string
		Start              bool
		Startdate          string
		Startup            string
		Storage            string
		Tablet             bool
		Tags               string
		Tdf                bool
		Template           bool
		Tpmstate0          string
		Unique             bool
		// Unused[n] string
		// Usb[n] string
		Vcpus int
		Vga   string
		// Virtio[n] string
		// Virtiofs[n] string
		Vmgenid        string
		Vmstatestorage string
		Watchdog       string
	}
}

func New(provider string) (ServerProvider, error) {
	switch provider {
	case "proxmox":
		return &ProxmoxProvider{}, nil
	default:
		return nil, errors.New(fmt.Sprintf("unknown provider: %s", provider))
	}
}

func (proxmox *ProxmoxProvider) CreateServer() (string, error) {
	return "", nil
}

func (proxmox *ProxmoxProvider) ReadServer() {

}

func (proxmox *ProxmoxProvider) UpdateServer() {

}

func (proxmox *ProxmoxProvider) DeleteServer() {

}

func (proxmox *ProxmoxProvider) Configure(host string, port int, username string, tokenId string, tokenSecret string) {
	proxmox.Host = host
	proxmox.Port = port
	proxmox.Username = username
	proxmox.TokenId = tokenId
	proxmox.TokenSecret = tokenSecret
}

