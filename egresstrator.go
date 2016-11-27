package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/urfave/cli"
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

func doRoutestration(containerId string, c *client.Client, env []string) bool {
	inspectedContainer, err := c.ContainerInspect(context.TODO(), containerId)
	if err != nil {
		log.Println(err)
		return false
	}

	enable := false
	for _, env := range inspectedContainer.Config.Env {
		if env == "ENABLE_EGRESSTRATOR=1" {
			enable = true
			break
		}
	}

	if !enable {
		log.Printf("Not enabling egresstrator for %v\n", containerId)
		return false
	}
	log.Printf("Enabling egresstrator for %v\n", containerId)

	config := container.Config{
		Image: "bonniernews/routestrator:latest",
		Cmd:   []string{"0.0.0.0/0"},
		Env:   env,
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
	app := cli.NewApp()
	app.Name = "egresstrator"
	app.Usage = "Set egress rules in network namespaces"
	app.Version = "0.0.1"
	app.Compiled = time.Now()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "consul, c",
			Value:  "127.0.0.1:8500",
			Usage:  "Consul address",
			EnvVar: "CONSUL_HTTP_ADDR",
		},
		cli.StringFlag{
			Name:   "consul-token, t",
			Usage:  "Consul token",
			EnvVar: "CONSUL_HTTP_TOKEN",
		},
		cli.StringFlag{
			Name:   "kv-path, k",
			Usage:  "Consul K/V path for egress ACL's",
			Value:  "egress/acl/",
			EnvVar: "CONSUL_PATH",
		},
		cli.StringFlag{
			Name:   "template, f",
			Usage:  "Custom consul template",
			EnvVar: "CONSUL_TEMPLATE",
		},
	}

	app.Action = func(c *cli.Context) error {
		log.Println("Starting egresstrator...")
		// handle args
		dockerEnv := []string{}
		dockerEnv = append(dockerEnv, fmt.Sprintf("CONSUL_HTTP_ADDR=%v", c.String("consul")))
		if c.GlobalIsSet("consul-token") {
			dockerEnv = append(dockerEnv, fmt.Sprintf("CONSUL_HTTP_TOKEM=%v", c.String("consul-token")))
		}
		dockerEnv = append(dockerEnv, fmt.Sprintf("CONSUL_PATH=%v", c.String("kv-path")))
		if c.GlobalIsSet("template") {
			dockerEnv = append(dockerEnv, fmt.Sprintf("CONSUL_TEMPLATE=%v", c.String("template")))
		}
		log.Println(dockerEnv)
		dockerClient, err := client.NewEnvClient()
		if err != nil {
			log.Fatal(err)
		}
		msg, errs := dockerClient.Events(context.Background(), types.EventsOptions{})
	Loop:
		for {
			select {
			case err := <-errs:
				if err != nil && err != io.EOF {
					log.Fatal(err)
				}
				break Loop
			case e := <-msg:
				log.Printf("Got event: %v  %v - %v\n", e.Type, e.Status, e.ID)
				if e.Status == "start" && e.Type == "container" {
					go doRoutestration(e.ID, dockerClient, dockerEnv)
				}
			}
		}
		return nil

	}
	app.Run(os.Args)
}
