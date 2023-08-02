package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aaletov/nats-chat/pkg/handlers"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	logDir := "/var/log/nats-chat"
	var err error
	if _, err = os.Stat(logDir); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(logDir, 0777); err != nil {
			log.Fatalf("Error creating log directory: %s\n", err)
		}
		if err = os.Chmod(logDir, 0777); err != nil {
			log.Fatalf("Error chmod: %s\n", err)
		}
	} else if err != nil {
		log.Fatal(err)
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("%d.%s", time.Now().Nanosecond(), "log"))
	var logFile *os.File
	if logFile, err = os.Create(logPath); err != nil {
		log.Fatalf("Error creating log file: %s\n", err)
	}
	defer logFile.Close()

	logger := logrus.New()
	logger.Out = logFile
	logger.SetFormatter(&nested.Formatter{
		HideKeys:    true,
		FieldsOrder: []string{"component", "method"},
	})

	var homeDir string

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
				Action: handlers.NewGenerateHandler(logger),
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
				Action: handlers.NewRunHandler(logger),
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
