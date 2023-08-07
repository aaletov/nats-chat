package main

import (
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	api "github.com/aaletov/nats-chat/api/generated"
	"github.com/aaletov/nats-chat/pkg/logger"
	"github.com/aaletov/nats-chat/pkg/natsdaemon"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	logger := logger.NewDefaultLogger()
	logger.SetLevel(logrus.DebugLevel)

	var (
		homeDir string
		err     error
	)

	if homeDir, err = os.UserHomeDir(); err != nil {
		logger.Fatalf("Unable to get user's home directory: %s", err)
	}
	natsDir := filepath.Join(homeDir, ".natschat")
	if _, err := os.Stat(natsDir); (err != nil) && (os.IsNotExist(err)) {
		if err := os.Mkdir(natsDir, 0700); err != nil {
			logger.Fatalf("error when create profile directory: %s", err)
		}
	}

	PROTOCOL := "unix"
	SOCKET := filepath.Join(natsDir, "socket/natschat.sock")

	lis, err := net.Listen(PROTOCOL, SOCKET)
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		logger.Fatalf("Got signal: %s", <-c)
	}()

	daemonServer := natsdaemon.NewDaemon(logger)
	logrus.RegisterExitHandler(func() {
		lis.Close()
		daemonServer.Shutdown()
	})
	s := grpc.NewServer()
	api.RegisterDaemonServer(s, daemonServer)

	logger.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		logger.Fatalf("failed to serve: %v", err)
	}
}
