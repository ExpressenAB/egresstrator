package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
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

func doRoutestration(containerId string, c *client.Client, dockerEnv []string, containerImage string) bool {
	inspectedContainer, err := c.ContainerInspect(context.TODO(), containerId)
	if err != nil {
		log.Println(err)
		return false
	}

	enable := false
	for _, env := range inspectedContainer.Config.Env {
		if env == "EGRESSTRATOR_ENABLE=1" {
			enable = true
		}
		if strings.HasPrefix(env, "EGRESSTRATOR_ACL") {
			dockerEnv = append(dockerEnv, env)
		}
	}

	if !enable {
		log.Printf("Not enabling egresstrator for %v\n", containerId)
		return false
	}
	log.Printf("Enabling egresstrator for %v\n", containerId)
	log.Println(dockerEnv)

	config := container.Config{
		Image: containerImage,
		Cmd:   []string{"set-egress"},
		Env:   dockerEnv,
	}
	hostConfig := container.HostConfig{
		CapAdd:      []string{"NET_ADMIN"},
		NetworkMode: container.NetworkMode(fmt.Sprintf("container:%v", containerId)),
		AutoRemove:  true,
		UsernsMode:  "host",
	}
	containerName := fmt.Sprintf("egresstrator-%v", containerId)
	createResp, err := c.ContainerCreate(context.Background(), &config, &hostConfig, &network.NetworkingConfig{}, containerName)
	if err != nil {
		log.Println(err)
		return false
	}
	err = c.ContainerStart(context.Background(), createResp.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Println(err)
		return false
	}
	// get container logs for 5 seconds...
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	reader, err := c.ContainerLogs(ctx, createResp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
	if err != nil {
		log.Fatal(err)
	}
	content, _ := ioutil.ReadAll(reader)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	log.Println(string(content))
	err = c.ContainerRemove(context.Background(), createResp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Fatal(err)
	}
	return true
}

func main() {
	app := cli.NewApp()
	app.Name = "egresstrator"
	app.Usage = "Set egress rules in network namespaces.\n   Enable egresstrator with EGRESSTRATOR_ENABLE=1 in your container.\n" +
		"   Specify egress rules with EGRESSTRATOR_ACL=myservice,otherservice"
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
		cli.StringFlag{
			Name:  "image, i",
			Usage: "Docker image name",
			Value: "expressenab/egresstrator:latest",
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
		dockerClient, err := client.NewEnvClient()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Pulling image: %v", c.String("image"))
		resp, err := dockerClient.ImagePull(context.Background(), c.String("image"), types.ImagePullOptions{})
		if err != nil {
			log.Fatal(err)
		}
		body, err := ioutil.ReadAll(resp)
		log.Printf("Image pulled: %v", string(body))
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
					go doRoutestration(e.ID, dockerClient, dockerEnv, c.String("image"))
				}
			}
		}
		return nil

	}
	app.Run(os.Args)
}
