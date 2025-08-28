package cli

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jl54/chunk-servers/internal/provider"
)

type CreateCmd struct {
	Cmd    flag.FlagSet
	Name   string
	Cores  int
	Memory int
}

type GetCmd struct {
	Cmd flag.FlagSet
	Id  int
}

type ListCmd struct {
	Cmd flag.FlagSet
}

type DeleteCmd struct {
	Cmd   flag.FlagSet
	Id    int
	Force bool
	Purge bool
}

func Handle() {
	if len(os.Args) < 2 {
		fmt.Println("expected 'create', 'get', 'list', 'update' or 'delete' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create":
		createCmd := &CreateCmd{}
		createCmd.HandleCreate()
	case "get":
		getCmd := &GetCmd{}
		getCmd.HandleGet()
	case "list":
		listCmd := &ListCmd{}
		listCmd.HandleList()
	case "delete":
		deleteCmd := &DeleteCmd{}
		deleteCmd.HandleDelete()
	default:
		fmt.Println("expected 'create', 'get', 'list', 'update' or 'delete' subcommands")
		os.Exit(1)
	}
}

func (createCmd *CreateCmd) HandleCreate() {
	createCmd.Cmd = *flag.NewFlagSet("create", flag.ExitOnError)
	createCmd.Cmd.StringVar(&createCmd.Name, "name", "", "the server name")
	createCmd.Cmd.IntVar(&createCmd.Cores, "cores", 2, "the amount of cores assigned to the server")
	createCmd.Cmd.IntVar(&createCmd.Memory, "memory", 2048, "the amount of memory assigned to the server")
	createCmd.Cmd.Parse(os.Args[2:])

	log.Printf("Creating server: {name=%s} {memory=%d} {cores=%d}\n", createCmd.Name, createCmd.Memory, createCmd.Cores)

	serverProvider, err := provider.New("proxmox")

	if err != nil {
		log.Fatalf("Could not create provider: %v", err)
	}

	err = serverProvider.ConfigureFromEnvironment()

	if err != nil {
		log.Fatal(err)
	}

	err = serverProvider.CreateServer(createCmd.Name, createCmd.Memory, createCmd.Cores)

	if err != nil {
		log.Fatal(err)
	}
}
func (getCmd *GetCmd) HandleGet() {
	getCmd.Cmd = *flag.NewFlagSet("create", flag.ExitOnError)
	getCmd.Cmd.IntVar(&getCmd.Id, "id", 0, "the id of the server to get")
	getCmd.Cmd.Parse(os.Args[2:])

	log.Printf("Getting server info: {vmid=%d}\n", getCmd.Id)

	serverProvider, err := provider.New("proxmox")

	if err != nil {
		log.Fatalf("Could not create provider: %v", err)
	}

	err = serverProvider.ConfigureFromEnvironment()

	if err != nil {
		log.Fatal(err)
	}

	serverProvider.ReadServer(getCmd.Id)

	if err != nil {
		log.Fatal(err)
	}
}
func (listCmd *ListCmd) HandleList() (any, error) {
	listCmd.Cmd = *flag.NewFlagSet("list", flag.ExitOnError)
	listCmd.Cmd.Parse(os.Args[2:])
	serverProvider, err := provider.New("proxmox")

	if err != nil {
		log.Fatalf("Could not create provider: %v\n", err)
	}

	err = serverProvider.ConfigureFromEnvironment()

	if err != nil {
		log.Fatalf("Missing configuration: %v", err)
	}

	serverProvider.ListServers()
	return struct{}{}, nil
}
func (deleteCmd *DeleteCmd) HandleDelete() (any, error) {
	deleteCmd.Cmd = *flag.NewFlagSet("delete", flag.ExitOnError)
	deleteCmd.Cmd.IntVar(&deleteCmd.Id, "id", 0, "the id of the server to get")
	deleteCmd.Cmd.BoolVar(&deleteCmd.Force, "force", false, "delete the server even when running")
	deleteCmd.Cmd.BoolVar(&deleteCmd.Purge, "purge", false, "purge the server from job configurations like backups")

	deleteCmd.Cmd.Parse(os.Args[2:])
	fmt.Printf("Deleting server: %d\n", deleteCmd.Id)

	serverProvider, err := provider.New("proxmox")

	if err != nil {
		log.Fatalf("Could not create provider: %v\n", err)
	}

	err = serverProvider.ConfigureFromEnvironment()

	if err != nil {
		log.Fatalf("Missing configuration: %v", err)
	}

	serverProvider.DeleteServer(deleteCmd.Id)

	return struct{}{}, nil
}
