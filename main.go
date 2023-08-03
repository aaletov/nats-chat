package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/aaletov/nats-chat/pkg/handlers"
	"github.com/urfave/cli/v2"
)

func main() {
	var (
		err     error
		homeDir string
	)

	if homeDir, err = os.UserHomeDir(); err != nil {
		log.Fatalf("Unable to get user's home directory: %s", err)
	}
	app := cli.App{
		Name:  "nats-chat",
		Usage: "Chat using nats",
		Action: func(cCtx *cli.Context) error {
			cli.ShowAppHelpAndExit(cCtx, 0)
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "log",
				Usage:    "Where to put log file",
				Required: false,
			},
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
				Action: handlers.NewGenerateHandler(),
			},
			{
				Name:  "address",
				Usage: "Print address of profile",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "profile",
						Usage: "Profile to use",
						Value: filepath.Join(homeDir, ".natschat"),
					},
				},
				Action: handlers.NewAddressHandler(),
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
						Name:     "recepient",
						Usage:    "Address of the recepient",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "nats-url",
						Usage:    "URL of nats instance",
						Required: true,
					},
				},
				Action: handlers.NewRunHandler(),
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
