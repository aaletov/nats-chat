package natsdaemon

import (
	"context"
	"fmt"
	"time"

	api "github.com/aaletov/nats-chat/api/generated"
	"github.com/hashicorp/go-multierror"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

type daemon struct {
	api.UnimplementedDaemonServer
	session *Session
	chat    *ChatConnection
	logger  *logrus.Entry
	nc      *nats.Conn
}

type ShutdownableDaemonServer interface {
	api.DaemonServer
	Shutdown()
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

	options := []nats.Option{nats.Timeout(30 * time.Second)}
	var nc *nats.Conn
	if nc, err = nats.Connect(req.NatsUrl, options...); err != nil {
		return nil, fmt.Errorf("error connecting to nats instance: %s", err)
	}
	d.nc = nc
	ll.Println("Connected to the nats server")

	if d.session, err = Online(d.logger.Logger, nc, req.SenderAddress); err != nil {
		return &emptypb.Empty{}, fmt.Errorf("failed to initialize session: %s", err)
	}
	ll.Debugf("Initialized new session: %s", req.NatsUrl)

	return &emptypb.Empty{}, nil
}

func (d *daemon) Offline(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	defer d.nc.Close()
	var err *multierror.Error
	err = multierror.Append(err, d.chat.Close())
	err = multierror.Append(d.session.Close())

	return &emptypb.Empty{}, err.ErrorOrNil()
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
	return &emptypb.Empty{}, d.chat.Close()
}

func (d *daemon) Send(srv api.Daemon_SendServer) error {
	return d.chat.Send(srv)
}

func (d *daemon) Shutdown() {
	if d.nc != nil {
		d.nc.Close()
	}
}
