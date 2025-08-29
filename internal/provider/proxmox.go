package provider

import (
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
	"strings"
	"time"
)

type ProxmoxProvider struct {
	Host         string
	Port         int
	Node         string
	Username     string
	TokenId      string
	TokenSecret  string
	BaseUrl      string
	TemplateVmId int
	Client       *http.Client
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

type VmArgs struct {
	Vmid      int    `json:"vmid,omitempty"`
	Memory    int    `json:"memory,omitempty"`
	Cores     int    `json:"cores,omitempty"`
	Net0      string `json:"net0,omitempty"`
	Scsihw    string `json:"scsihw,omitempty"`
	Name      string `json:"name,omitempty"`
	Scsi0     string `json:"scsi0,omitempty"`
	Ide2      string `json:"ide2,omitempty"`
	Cicustom  string `json:"cicustom,omitempty"`
	Ipconfig0 string `json:"ipconfig0,omitempty"`
	Boot      string `json:"boot,omitempty"`
	Agent     string `json:"agent,omitempty"`
	Serial0   string `json:"serial0,omitempty"`
	Vga       string `json:"vga,omitempty"`
}

type VmResponse struct {
	Vmid   int
	Status string
}

type VmGetResponse struct {
	Data VmResponse
}

type VmListResponse struct {
	Data []VmResponse
}

type NextIdResponse struct {
	Data string
}

type VmCloneArgs struct {
	Newid int    `json:"newid"`
	Node  string `json:"node"`
	Vmid  int    `json:"vmid"`
	Full  bool   `json:"full,omitempty"`
}

func (proxmox *ProxmoxProvider) CreateTemplateVm(templateVmId int) error {
	log.Println("Creating template VM ...")
	// 1. create the vm
	data := VmArgs{
		Vmid:   templateVmId,
		Memory: 1024,
		Net0:   "virtio,bridge=vmbr0",
		Scsihw: "virtio-scsi-pci",
		Name:   "NotAnakin",
	}

	err := proxmox.createVm(data)

	if err != nil {
		return err
	}

	// 2. import the disk
	data = VmArgs{
		Vmid:  templateVmId,
		Scsi0: "local-lvm:0,import-from=local:import/novle-server-cloudimg-amd64.qcow2",
	}

	err = proxmox.updateVm(data)

	if err != nil {
		return err
	}

	// 3. add cloud init
	data = VmArgs{
		Vmid:      templateVmId,
		Ide2:      "local-lvm:cloudinit",
		Cicustom:  "user=local:snippets/userconfig.yml",
		Ipconfig0: "ip=dhcp",
		Boot:      "order=scsi0",
		Agent:     "enabled=1,type=virtio",
		Serial0:   "socket",
		Vga:       "serial0",
	}

	err = proxmox.updateVm(data)

	if err != nil {
		return err
	}

	// 4. convert to template
	err = proxmox.convertToTemplate(templateVmId)

	if err != nil {
		log.Fatalf("proxmox request failed: %v", err)
	}

	return nil
}

func (proxmox *ProxmoxProvider) CreateServer(name string, memory, cores int) error {
	vmInfo, err := proxmox.ReadServer(proxmox.TemplateVmId)

	if err != nil {
		return err
	}

	if vmInfo.Data.Status == "missing" {
		log.Println("Template VM is missing ...")

		err = proxmox.CreateTemplateVm(proxmox.TemplateVmId)

		if err != nil {
			return err
		}
	}

	nextId, err := proxmox.getNextVmId()

	if err != nil {
		return err
	}

	cloneData := VmCloneArgs{
		Newid: nextId,
		Node:  proxmox.Node,
		Vmid:  proxmox.TemplateVmId,
		Full:  true,
	}

	err = proxmox.cloneVm(cloneData)

	if err != nil {
		return err
	}

	updateData := VmArgs{
		Vmid:   nextId,
		Name:   name,
		Memory: memory,
		Cores:  cores,
	}

	err = proxmox.updateVm(updateData)

	if err != nil {
		return err
	}

	err = proxmox.startVm(nextId)

	return err
}

func (proxmox *ProxmoxProvider) ReadServer(id int) (VmGetResponse, error) {
	log.Printf("Getting VM Information for %d ...", id)
	vmPath := fmt.Sprintf("%s/nodes/%s/qemu/%d/status/current", proxmox.BaseUrl, proxmox.Node, id)
	var resObject VmGetResponse
	err := proxmox.Get(vmPath, &resObject)

	if err != nil {
		if err.Error() == "VM not found" {
			resObject.Data.Vmid = id
			resObject.Data.Status = "missing"
			return resObject, nil
		}

		return VmGetResponse{}, err
	}

	return resObject, nil
}

func (proxmox *ProxmoxProvider) ListServers() (VmListResponse, error) {
	log.Println("Listing servers ...")
	listPath := fmt.Sprintf("%s/nodes/%s/qemu", proxmox.BaseUrl, proxmox.Node)
	var resObject VmListResponse
	err := proxmox.Get(listPath, &resObject)

	if err != nil {
		return VmListResponse{}, err
	}

	return resObject, nil
}

func (proxmox *ProxmoxProvider) UpdateServer() {
	// Todo: ...
}

func (proxmox *ProxmoxProvider) DeleteServer(vmid int, force, purge bool) error {
	log.Printf("Deleting server %d ...\n", vmid)
	if force == true {
		err := proxmox.stopVm(vmid)

		if err != nil {
			return err
		}
	}

	deletePath := fmt.Sprintf("%s/nodes/%s/qemu/%d", proxmox.BaseUrl, proxmox.Node, vmid)

	if purge == true {
		deletePath = fmt.Sprintf("%s?purge=1&destroy-unreferenced-disks=1", deletePath)
	} else {
		deletePath = fmt.Sprintf("%s?destroy-unreferenced-disks=1", deletePath)
	}

	err := proxmox.Delete(deletePath)

	return err
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

	proxmox.TemplateVmId, err = strconv.Atoi(os.Getenv("PVE_TEMPLATE_ID"))

	if err != nil {
		proxmox.TemplateVmId = 0
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

	if proxmox.TemplateVmId == 0 {
		return errors.New("Missing proxmox template id")
	}

	proxmox.BaseUrl = fmt.Sprintf("https://%s:%d/api2/json", proxmox.Host, proxmox.Port)

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

	if res.StatusCode >= 500 {
		var resBody []byte
		resBody, err = io.ReadAll(res.Body)
		defer res.Body.Close()

		if strings.Contains(string(resBody), "conf' does not exist") {
			return errors.New("VM not found")
		}

		log.Println(string(resBody))
		return errors.New("500 Internal Server error")
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

func (proxmox *ProxmoxProvider) Post(path string, reqData any, resData any) error {
	body, err := proxmox.getJsonBodyFromData(reqData)

	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, path, body)

	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Authorization: PVEAPIToken=%s=%s", proxmox.TokenId, proxmox.TokenSecret))
	req.Header.Add("Content-Type", "application/json")

	res, err := proxmox.Client.Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	var resJsonString []byte
	resJsonString, err = io.ReadAll(res.Body)

	return proxmox.getDataFromJsonResponse(resJsonString, resData)
}

func (proxmox *ProxmoxProvider) Delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, path, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Authorization: PVEAPIToken=%s=%s", proxmox.TokenId, proxmox.TokenSecret))
	req.Header.Add("Accept", "application/json")

	res, err := proxmox.Client.Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	return nil
}

func (proxmox *ProxmoxProvider) getJsonBodyFromData(data any) (*bytes.Buffer, error) {
	json, err := json.Marshal(data)

	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(json), nil
}

func (proxmox *ProxmoxProvider) getDataFromJsonResponse(jsonData []byte, data any) error {
	return json.Unmarshal(jsonData, data)
}

func (proxmox *ProxmoxProvider) createVm(data VmArgs) error {
	createPath := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu", proxmox.Host, proxmox.Port, proxmox.Node)
	var resData ProxmoxTaskResponse
	err := proxmox.Post(
		createPath,
		data,
		&resData,
	)

	if err != nil {
		return err
	}

	status := "running"
	upid := resData.Data

	for status == "running" {
		time.Sleep(1 * time.Second)
		status = proxmox.GetTaskStatus(upid)
	}

	return nil
}

func (proxmox *ProxmoxProvider) updateVm(data VmArgs) error {
	log.Printf("Updating VM %d ...\n", data.Vmid)
	configPath := fmt.Sprintf("%s/nodes/%s/qemu/%d/config", proxmox.BaseUrl, proxmox.Node, data.Vmid)
	var resData ProxmoxTaskResponse
	err := proxmox.Post(configPath, data, &resData)

	if err != nil {
		return err
	}

	upid := resData.Data
	status := "running"

	for status == "running" {
		time.Sleep(1 * time.Second)
		status = proxmox.GetTaskStatus(upid)
	}

	return nil
}

func (proxmox *ProxmoxProvider) convertToTemplate(vmid int) error {
	log.Printf("Converting VM %d to template ...\n", vmid)
	convertToTemplatePath := fmt.Sprintf("%s/nodes/%s/qemu/%d/template", proxmox.BaseUrl, proxmox.Node, vmid)
	var resData ProxmoxTaskResponse
	err := proxmox.Post(convertToTemplatePath, nil, &resData)

	if err != nil {
		return err
	}

	upid := resData.Data
	status := "running"

	for status == "running" {
		time.Sleep(1 * time.Second)
		status = proxmox.GetTaskStatus(upid)
	}

	return nil
}

func (proxmox *ProxmoxProvider) getNextVmId() (int, error) {
	log.Println("Getting next VM id ...")
	path := fmt.Sprintf("%s/cluster/nextid", proxmox.BaseUrl)

	var resObject NextIdResponse
	err := proxmox.Get(path, &resObject)

	if err != nil {
		return 0, err
	}

	vmid, err := strconv.Atoi(resObject.Data)

	if err != nil {
		return 0, err
	}

	return vmid, nil
}

func (proxmox *ProxmoxProvider) cloneVm(data VmCloneArgs) error {
	log.Println("Cloning Template VM ...")
	var resData ProxmoxTaskResponse
	clonePath := fmt.Sprintf("%s/nodes/%s/qemu/%d/clone", proxmox.BaseUrl, proxmox.Node, proxmox.TemplateVmId)
	err := proxmox.Post(clonePath, data, &resData)

	if err != nil {
		return err
	}

	upid := resData.Data
	status := "running"

	for status == "running" {
		time.Sleep(1 * time.Second)
		status = proxmox.GetTaskStatus(upid)
	}

	return nil
}

func (proxmox *ProxmoxProvider) startVm(vmid int) error {
	log.Printf("Starting VM %d ...\n", vmid)
	startData := struct {
		Vmid int    `json:"vmid"`
		Node string `json:"node"`
	}{
		Vmid: vmid,
		Node: proxmox.Node,
	}
	var resData ProxmoxTaskResponse
	startPath := fmt.Sprintf("%s/nodes/%s/qemu/%d/status/start", proxmox.BaseUrl, proxmox.Node, vmid)
	err := proxmox.Post(startPath, startData, &resData)

	if err != nil {
		return err
	}

	upid := resData.Data
	status := "running"

	for status == "running" {
		time.Sleep(1 * time.Second)
		status = proxmox.GetTaskStatus(upid)
	}

	return nil
}

func (proxmox *ProxmoxProvider) stopVm(vmid int) error {
	log.Printf("Stopping VM %d ...\n", vmid)
	stopData := struct {
		Vmid int    `json:"vmid"`
		Node string `json:"node"`
	}{
		Vmid: vmid,
		Node: proxmox.Node,
	}
	var resData ProxmoxTaskResponse
	stopPath := fmt.Sprintf("%s/nodes/%s/qemu/%d/status/stop", proxmox.BaseUrl, proxmox.Node, vmid)
	err := proxmox.Post(stopPath, stopData, &resData)

	if err != nil {
		return err
	}

	upid := resData.Data
	status := "running"

	for status == "running" {
		time.Sleep(1 * time.Second)
		status = proxmox.GetTaskStatus(upid)
	}

	return nil
}
