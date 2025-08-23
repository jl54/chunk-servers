package provider

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

type ProxmoxProvider struct {
	Host        string
	Port        int
	Node        string
	Username    string
	TokenId     string
	TokenSecret string
}

// hostpci0  -> '0000:01:00,rombar=0,x-vga=1'
type HostpciConfig struct {
	VendorId string
	Rombar   bool
	XVga     bool
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
		Hostpcis    map[string]HostpciConfig
		Hotplug     string
		Hugepages   string
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

func (proxmox *ProxmoxProvider) CreateServer() (string, error) {
	fmt.Println("creating server...")
	return "", nil
}

func (proxmox *ProxmoxProvider) ReadServer() {

}

func (proxmox *ProxmoxProvider) ListServers() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu", proxmox.Host, proxmox.Port, proxmox.Node), nil)

	if err != nil {
		log.Fatalf("proxmox request failed: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Authorization: PVEAPIToken=%s=%s", proxmox.TokenId, proxmox.TokenSecret))
	resp, err := client.Do(req)

	if err != nil {
		log.Fatalf("proxmox request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)
	scanner := bufio.NewScanner(resp.Body)

	for i := 0; scanner.Scan() && i < 5; i++ {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("proxmox request failed: %v", err)
	}
}

func (proxmox *ProxmoxProvider) UpdateServer() {

}

func (proxmox *ProxmoxProvider) DeleteServer() {

}

func (proxmox *ProxmoxProvider) ConfigureFromEnvironment() error {
	proxmox.Host = os.Getenv("PVE_HOST")
	proxmox.TokenId = os.Getenv("PVE_TOKEN_ID")
	proxmox.TokenSecret = os.Getenv("PVE_TOKEN_SECRET")
	proxmox.Node = os.Getenv("PVE_NODE")

	var err error
	proxmox.Port, err = strconv.Atoi(os.Getenv("PVE_PORT"))

	if err != nil {
		proxmox.Port = 0
	}

	if proxmox.Host == "" {
		return errors.New("Missing proxmox host")
	}

	if proxmox.TokenId == "" {
		return errors.New("Missing proxmox token id")
	}

	if proxmox.TokenSecret == "" {
		return errors.New("Missing proxmox token secret")
	}

	if proxmox.Node == "" {
		return errors.New("Missing proxmox node")
	}

	if proxmox.Port == 0 {
		return errors.New("Missing proxmox port")
	}

	return nil
}

func (proxmox *ProxmoxProvider) Configure(host string, port int, username string, tokenId string, tokenSecret string) {
	proxmox.Host = host
	proxmox.Port = port
	proxmox.Username = username
	proxmox.TokenId = tokenId
	proxmox.TokenSecret = tokenSecret
}
