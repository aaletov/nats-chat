package main

import (
	"os"
	"path/filepath"

	"github.com/aaletov/nats-chat/pkg/clihandler"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&nested.Formatter{
		HideKeys:    true,
		FieldsOrder: []string{"component", "method"},
	})

	var (
		err     error
		homeDir string
	)

	if homeDir, err = os.UserHomeDir(); err != nil {
		logger.Fatalf("Error getting user home: %s", err)
	}
	natsDir := filepath.Join(homeDir, ".natschat")

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
						Name:     "out",
						Usage:    "Where to put generated profile",
						Required: false,
						Value:    natsDir,
					},
				},
				Action: clihandler.NewGenerateHandler(logger),
			},
			{
				Name:  "address",
				Usage: "Print address of profile",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "profile",
						Usage:    "Profile to use",
						Required: false,
						Value:    natsDir,
					},
				},
				Before: clihandler.CheckProfileDir,
				Action: clihandler.NewAddressHandler(logger),
			},
			{
				Name:  "online",
				Usage: "Go online",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "profile",
						Usage:    "Path to the nats-chat profile",
						Required: false,
						Value:    natsDir,
					},
					&cli.StringFlag{
						Name:     "nats-url",
						Usage:    "URL of nats instance",
						Required: true,
					},
				},
				Before: clihandler.CheckProfileDir,
				Action: clihandler.NewOnlineHandler(logger),
			},
			{
				Name:   "offline",
				Usage:  "Go offline",
				Action: clihandler.NewOfflineHandler(logger),
			},
			{
				Name:  "createchat",
				Usage: "Start a chat",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "recepient",
						Usage:    "Address of the recepient",
						Required: true,
					},
				},
				Action: clihandler.NewCreateChatHandler(logger),
			},
			{
				Name:  "rmchat",
				Usage: "Close chat",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "recepient",
						Usage:    "Address of the recepient",
						Required: true,
					},
				},
				Action: clihandler.NewRmChatHandler(logger),
			},
			{
				Name:   "openchat",
				Usage:  "Open chat",
				Action: clihandler.NewOpenChatHandler(logger),
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}
