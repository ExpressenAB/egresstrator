package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/net/context"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"
)

type Event struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

type Container struct {
	Id    string
	Pid   int
	Image string
}

func doRoutestration(containerId string, c *client.Client, gw string) bool {
	inspectedContainer, err := c.ContainerInspect(context.TODO(), containerId)
	if err != nil {
		log.Println(err)
		return false
	}

	enable := false
	for _, env := range inspectedContainer.Config.Env {
		if env == "ENABLE_ROUTESTRATOR=1" {
			enable = true
			break
		}
	}

	if !enable {
		log.Printf("Not enabling routestrator for %v\n", containerId)
		return false
	}
	log.Printf("Enabling routestrator for %v\n", containerId)

	config := container.Config{
		Image: "bonniernews/routestrator:latest",
		Cmd:   []string{gw},
	}
	hostConfig := container.HostConfig{
		CapAdd:      []string{"NET_ADMIN"},
		NetworkMode: container.NetworkMode(fmt.Sprintf("container:%v", containerId)),
	}
	containerName := fmt.Sprintf("routestrator-%v", containerId)
	createResp, err := c.ContainerCreate(context.TODO(), &config, &hostConfig, &network.NetworkingConfig{}, containerName)
	if err != nil {
		log.Println(err)
		return false
	}
	err = c.ContainerStart(context.TODO(), createResp.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Println(err)
		return false
	}

	return true
}

func main() {
	gw := os.Args[1]
	log.Printf("Will set gateway to %v\n", gw)

	dockerClient, err := client.NewEnvClient()
	if err != nil {
		log.Fatal(err)
	}
	reader, err := dockerClient.Events(context.TODO(), types.EventsOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	d := json.NewDecoder(reader)
	for {
		var event Event
		if err := d.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			log.Println(err)
		}
		if event.Status == "start" && event.Type == "container" {
			go doRoutestration(event.Id, dockerClient, gw)
		}
	}
}
