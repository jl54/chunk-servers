package provider

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type ProxmoxProvider struct {
	Host        string
	Port        int
	Node        string
	Username    string
	TokenId     string
	TokenSecret string
	Client      *http.Client
}

type ProxmoxServerOptions struct {
	Name    string
	Memory  int
	Cpu     string
	Cores   int
	Scsihw  string
	Ide2    string
	Scsi0   string
	Network string
}

type ProxmoxTaskResponse struct {
	Data string
}

type ProxmoxTaskGetResponse struct {
	Data struct {
		User       string
		Pid        uint32
		Starttime  uint32
		Tokenid    string
		Type       string
		Node       string
		Exitstatus string
		Id         string
		Pstart     uint32
		Upid       string
		Status     string
	}
}

func (proxmox *ProxmoxProvider) CreateTemplateVm(templateVmId int) {
	// 1. create the vm
	data := make(map[string]any)
	data["vmid"] = templateVmId
	data["memory"] = 1024
	data["net0"] = "virtio,bridge=vmbr0"
	data["scsihw"] = "virtio-scsi-pci"
	status := "running"
	upid, err := proxmox.Post(fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu", proxmox.Host, proxmox.Port, proxmox.Node), data)

	for status == "running" {
		time.Sleep(1 * time.Second)
		status = proxmox.GetTaskStatus(upid)
	}

	if err != nil {
		log.Fatalf("Could not create template vm: %v", err)
	}

	// 2. import the disk
	data = make(map[string]any)
	data["vmid"] = templateVmId
	data["scsi0"] = "local-lvm:0,import-from=local:import/novle-server-cloudimg-amd64.qcow2"
	data["ide2"] = "local-lvm:cloudinit"
	data["boot"] = "order=scsi0"
	data["cicustom"] = "user=local:snippets/userconfig.yml"
	data["ipconfig0"] = "ip=dhcp"
	status = "running"
	upid, err = proxmox.Post(fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu/%d/config", proxmox.Host, proxmox.Port, proxmox.Node, templateVmId), data)

	if err != nil {
		log.Fatalf("proxmox request failed: %v", err)
	}

	for status == "running" {
		time.Sleep(1 * time.Second)
		status = proxmox.GetTaskStatus(upid)
	}

	// 3. convert to template
	status = "running"
	upid, err = proxmox.Post(fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu/%d/template", proxmox.Host, proxmox.Port, proxmox.Node, templateVmId), nil)

	if err != nil {
		log.Fatalf("proxmox request failed: %v", err)
	}

	for status == "running" {
		time.Sleep(1 * time.Second)
		status = proxmox.GetTaskStatus(upid)
	}
}

func (proxmox *ProxmoxProvider) CreateServer() (string, error) {
	proxmox.CreateTemplateVm(9001)
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

func (proxmox *ProxmoxProvider) DeleteServer(vmid int) {
	err := proxmox.Delete(fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu/%d", proxmox.Host, proxmox.Port, proxmox.Node, vmid))

	if err != nil {
		log.Fatalf("Error deleting server: %v", err)
	}
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

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	proxmox.Client = &http.Client{
		Transport: transport,
	}

	return nil
}

func (proxmox *ProxmoxProvider) GetTaskStatus(upid string) string {
	var taskInfo ProxmoxTaskGetResponse
	path := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/tasks/%s/status", proxmox.Host, proxmox.Port, proxmox.Node, upid)

	err := proxmox.Get(path, &taskInfo)

	if err != nil {
		log.Fatalf("Errored getting task info: %v", err)
	}

	log.Println(taskInfo.Data.Status)
	return taskInfo.Data.Status
}

func (proxmox *ProxmoxProvider) Get(path string, resObject any) error {
	req, err := http.NewRequest(http.MethodGet, path, nil)

	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Authorization: PVEAPIToken=%s=%s", proxmox.TokenId, proxmox.TokenSecret))
	req.Header.Add("Accept", "application/json")
	var res *http.Response
	res, err = proxmox.Client.Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	var resBody []byte
	resBody, err = io.ReadAll(res.Body)
	err = json.Unmarshal(resBody, resObject)

	if err != nil {
		return err
	}

	return nil
}

func (proxmox *ProxmoxProvider) Post(path string, data any) (string, error) {
	var err error
	var body io.Reader
	var req *http.Request
	var jsonData []byte
	jsonData, err = json.Marshal(data)

	if err != nil {
		return "", err
	}

	body = bytes.NewBuffer(jsonData)
	req, err = http.NewRequest(http.MethodPost, path, body)

	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Authorization: PVEAPIToken=%s=%s", proxmox.TokenId, proxmox.TokenSecret))
	req.Header.Add("Content-Type", "application/json")
	var res *http.Response
	res, err = proxmox.Client.Do(req)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	log.Println(res.StatusCode)
	var resBody []byte
	resBody, err = io.ReadAll(res.Body)
	log.Println(string(resBody))

	var resJson ProxmoxTaskResponse
	err = json.Unmarshal(resBody, &resJson)

	if err != nil {
		log.Fatalf("Errored unmarshaling json: %v", err)
	}

	log.Println(resJson.Data)

	// var resJson map[string]json.RawMessage
	// err = json.Unmarshal(resBody, &resJson)
	//
	// if err != nil {
	// 	return "", err
	// }
	//
	// var resData []byte
	// err = json.Unmarshal(resJson["data"], &resData)
	//
	// if err != nil {
	// 	return "", err
	// }
	//
	// fmt.Println(string(resData))
	return resJson.Data, nil

}

func (proxmox *ProxmoxProvider) Delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, path, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Authorization: PVEAPIToken=%s=%s", proxmox.TokenId, proxmox.TokenSecret))
	req.Header.Add("Accept", "application/json")

	_, err = proxmox.Client.Do(req)

	if err != nil {
		return err
	}

	return nil
}
