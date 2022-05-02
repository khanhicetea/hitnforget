package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"

	"github.com/khanhicetea/hitnforget/server"
	"github.com/khanhicetea/hitnforget/worker"
	cli "github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:        "HitNForget",
		Description: "HTTP Later Server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "server",
				Aliases: []string{"s"},
				Usage:   "Run HTTP queue server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "bind",
						Usage: "Binding address",
						Value: "127.0.0.1",
					},
					&cli.StringFlag{
						Name:  "port",
						Usage: "Binding port",
						Value: "3333",
					},
				},
				Action: func(c *cli.Context) error {
					bind := c.Value("bind").(string)
					port := c.Value("port").(string)
					fmt.Printf("Running queue server on %s:%s ...", bind, port)
					http.ListenAndServe(net.JoinHostPort(bind, port), server.HTTPHandler())
					return nil
				},
			},
			{
				Name:    "worker",
				Aliases: []string{"w"},
				Usage:   "Run worker",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "working_queue",
						Usage: "Working queue name",
						Value: "default",
					},
					&cli.StringFlag{
						Name:  "failed_queue",
						Usage: "Next failed queue",
						Value: "",
					},
				},
				Action: func(c *cli.Context) error {
					workingQueue := c.Value("working_queue").(string)
					failedQueue := c.Value("failed_queue").(string)
					fmt.Printf("Running worker on %s and fallback to %s ...", workingQueue, failedQueue)
					worker.Worker(rand.Intn(100), workingQueue, failedQueue)
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
