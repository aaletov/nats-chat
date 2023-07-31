package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aaletov/nats-chat/pkg/handlers"
	"github.com/urfave/cli/v2"
)

func main() {
	var (
		homeDir string
		err     error
	)

	if homeDir, err = os.UserHomeDir(); err != nil {
		log.Fatalf("Unable to get user's home directory: %s", err)
	}
	fmt.Println(homeDir)
	app := cli.App{
		Name:  "nats-chat",
		Usage: "Chat using nats",
		Action: func(cCtx *cli.Context) error {
			cli.ShowAppHelpAndExit(cCtx, 0)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "generate",
				Usage: "Generate key pair",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "out",
						Usage: "Where to put generated profile",
						Value: homeDir,
					},
				},
				Action: handlers.GenerateHandler,
			},
			{
				Name:  "run",
				Usage: "Start a chat",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "profile",
						Usage: "Path to the nats-chat profile",
						Value: filepath.Join(homeDir, ".natschat"),
					},
					&cli.StringFlag{
						Name:     "recepient-key",
						Usage:    "Public key of the recepient",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "nats-url",
						Usage:    "URL of nats instance",
						Required: true,
					},
				},
				Action: handlers.RunHandler,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
