package main

import (
	"bytes"
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
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/urfave/cli"
)

var Version string

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

func doEgresstration(containerId string, c *client.Client, dockerEnv []string, containerImage string, mode string) bool {
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
		log.Printf("Egresstrator not enabled on %v\n", containerId)
		return false
	}
	log.Printf("%s egress rules for %v\n", strings.ToTitle(mode), containerId)
	log.Println(dockerEnv)

	config := container.Config{
		Image: containerImage,
		Cmd:   []string{mode + "-egress"},
		Env:   dockerEnv,
	}
	hostConfig := container.HostConfig{
		CapAdd:      []string{"NET_ADMIN"},
		NetworkMode: container.NetworkMode(fmt.Sprintf("container:%v", containerId)),
		AutoRemove:  false,
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

	// Get container logs for MAX 30 seconds or until container stops
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	reader, err := c.ContainerLogs(ctx, createResp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	var stdout, stderr bytes.Buffer
	if inspectedContainer.Config.Tty {
		_, err = io.Copy(&stdout, reader)
	} else {
		_, err = stdcopy.StdCopy(&stdout, &stderr, reader)
	}

	if ctx.Err() == context.DeadlineExceeded {
		log.Println("Egresstrator container not stopping. Shutting down")
		err := c.ContainerKill(context.Background(), createResp.ID, "KILL")
		if err != nil {
			log.Println(err)
		}
	}
	log.Printf("%s", stderr.String())
	log.Printf("%s", stdout.String())

	err = c.ContainerRemove(context.Background(), createResp.ID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Fatal(err)
	}
	return true
}

func imagePull(dockerClient *client.Client, image string) {
	log.Printf("Pulling image: %v", image)
	resp, err := dockerClient.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(resp)
	log.Printf("Docker: %v", string(body))
}

func imageLoad(dockerClient *client.Client) {
	log.Println("Loading embedded image: egresstrator:latest")

	data, err := Asset("egresstrator.tar")
	if err != nil {
		log.Fatal("Embedded image not found")
	}

	image := bytes.NewReader(data)

	resp, err := dockerClient.ImageLoad(context.Background(), image, true)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	log.Printf(string(body))
}

func initApp(c *cli.Context) (*client.Client, []string, string) {
	log.Println("Starting egresstrator...")
	// handle args
	dockerEnv := []string{}
	dockerEnv = append(dockerEnv, fmt.Sprintf("CONSUL_HTTP_ADDR=%v", c.GlobalString("consul")))
	if c.GlobalIsSet("consul-token") {
		dockerEnv = append(dockerEnv, fmt.Sprintf("CONSUL_HTTP_TOKEN=%v", c.GlobalString("consul-token")))
	}
	dockerEnv = append(dockerEnv, fmt.Sprintf("CONSUL_PATH=%v", c.GlobalString("kv-path")))
	if c.GlobalIsSet("template") {
		dockerEnv = append(dockerEnv, fmt.Sprintf("CONSUL_TEMPLATE=%v", c.GlobalString("template")))
	}
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		log.Fatal(err)
	}

	image := "egresstrator:latest"
	if c.GlobalIsSet("image") {
		imagePull(dockerClient, c.GlobalString("image"))
		image = c.GlobalString("image")
	} else {
		imageLoad(dockerClient)
	}

	return dockerClient, dockerEnv, image
}

func doSetClearCommand(c *cli.Context) error {

	containerID := ""
	command := c.Command.Name

	if c.Bool("all") {
		log.Println("Execute on all running containers")
	} else if len(c.Args()) == 0 {
		cli.ShowCommandHelp(c, command)
		return cli.NewExitError("Error: Container ID not specified as argument", 1)
	} else {
		containerID = strings.ToLower(c.Args().Get(0))
	}

	dockerClient, dockerEnv, image := initApp(c)

	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	if c.Bool("all") {
		for _, container := range containers {
			doEgresstration(container.ID, dockerClient, dockerEnv, image, command)
		}
	} else {
		doEgresstration(containerID, dockerClient, dockerEnv, image, command)
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "egresstrator"
	app.Usage = "Set egress rules in network namespaces.\n   Enable egresstrator with EGRESSTRATOR_ENABLE=1 in your container.\n" +
		"   Specify egress rules with EGRESSTRATOR_ACL=myservice,otherservice"
	app.Version = Version
	app.Compiled = time.Now()

	app.Commands = []cli.Command{
		{
			Name:  "set",
			Usage: "Set egress rules on specified container",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "Set egress rules on all running containers",
				},
			},
			Action: func(c *cli.Context) error {
				return doSetClearCommand(c)
			},
		},
		{
			Name:  "clear",
			Usage: "Clear egress rules on specified container",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "all, a",
					Usage: "Clear egress rules on all running containers",
				},
			},
			Action: func(c *cli.Context) error {
				return doSetClearCommand(c)
			},
		},
	}

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
		},
	}

	app.Action = func(c *cli.Context) error {
		dockerClient, dockerEnv, image := initApp(c)
		log.Println("Listening on docker events")
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
					go doEgresstration(e.ID, dockerClient, dockerEnv, image, "set")
				}
			}
		}
		return nil

	}
	app.Run(os.Args)
}
