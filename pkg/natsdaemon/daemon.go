package natsdaemon

import (
	"context"
	"fmt"

	api "github.com/aaletov/nats-chat/api/generated"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

type daemon struct {
	api.UnimplementedDaemonServer
	session *Session
	chat    *Chat
	logger  *logrus.Entry
}

type ShutdownableDaemonServer interface {
	api.DaemonServer
	Shutdown() error
}

func NewDaemon(logger *logrus.Logger) ShutdownableDaemonServer {
	return &daemon{
		logger: logger.WithFields(logrus.Fields{
			"component": "DaemonServer",
		}),
	}
}

func (d *daemon) Online(ctx context.Context, req *api.OnlineRequest) (*emptypb.Empty, error) {
	ll := d.logger.WithFields(logrus.Fields{
		"method": "Online",
	})
	ll.Debugf("Processing request: %s", req)
	var err error

	if d.session, err = Online(d.logger.Logger, req.NatsUrl, req.SenderAddress); err != nil {
		return &emptypb.Empty{}, fmt.Errorf("failed to initialize session: %s", err)
	}
	ll.Debugf("Initialized new session: %s", req.NatsUrl)

	return &emptypb.Empty{}, nil
}

func shutdownDaemon(d *daemon) error {
	var err *multierror.Error

	if d.session != nil {
		err = multierror.Append(d.session.Close())
		d.session = nil
	}

	return err.ErrorOrNil()
}

func (d *daemon) Offline(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, shutdownDaemon(d)
}

func (d *daemon) CreateChat(ctx context.Context, req *api.ChatRequest) (*emptypb.Empty, error) {
	ll := d.logger.WithFields(logrus.Fields{
		"method": "CreateChat",
	})
	var err error
	if d.chat, err = d.session.Dial(req.RecepientAddress); err != nil {
		return &emptypb.Empty{}, err
	}
	ll.Debugf("Dialed successfully: %s", req.RecepientAddress)
	return &emptypb.Empty{}, nil
}

func (d *daemon) DeleteChat(ctx context.Context, req *api.ChatRequest) (*emptypb.Empty, error) {
	ll := d.logger.WithFields(logrus.Fields{
		"method": "DeleteChat",
	})
	if d.chat != nil {
		err := d.session.CloseChat()
		d.chat = nil
		return &emptypb.Empty{}, err
	}
	ll.Debugf("Chat does not exist: %s", req.RecepientAddress)
	return &emptypb.Empty{}, nil
}

func (d *daemon) Send(srv api.Daemon_SendServer) error {
	return d.chat.Send(srv)
}

func (d *daemon) Shutdown() error {
	return shutdownDaemon(d)
}
