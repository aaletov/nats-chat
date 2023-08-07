package natscli

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"

	api "github.com/aaletov/nats-chat/api/generated"
	"github.com/aaletov/nats-chat/pkg/fs"
	"github.com/aaletov/nats-chat/pkg/profile"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ConnectDaemon(cCtx *cli.Context) (*grpc.ClientConn, error) {
	var (
		homeDir string
		err     error
	)

	if homeDir, err = os.UserHomeDir(); err != nil {
		return nil, fmt.Errorf("Unable to get user's home directory: %s", err)
	}
	PROTOCOL := "unix"
	SOCKET := filepath.Join(homeDir, ".natschat/socket/natschat.sock")
	dialOption := grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return net.Dial(PROTOCOL, s)
	})
	secOption := grpc.WithTransportCredentials(insecure.NewCredentials())

	conn, err := grpc.Dial(SOCKET, dialOption, secOption)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

type CliDaemonHandler func(*cli.Context, *logrus.Entry, api.DaemonClient) error

func WrapCliDaemonHandler(handler CliDaemonHandler) CliHandler {
	return func(cCtx *cli.Context, ll *logrus.Entry) error {
		var err error
		var daemonConnection *grpc.ClientConn
		if daemonConnection, err = ConnectDaemon(cCtx); err != nil {
			return fmt.Errorf("failed to initialize daemonConnection: %s", err)
		}
		defer daemonConnection.Close()
		ll.Println("Connected to daemon")
		daemonClient := api.NewDaemonClient(daemonConnection)
		return handler(cCtx, ll, daemonClient)
	}
}

type CliHandler func(*cli.Context, *logrus.Entry) error

func WrapCliHandler(handler CliHandler, logger *logrus.Entry) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		return handler(ctx, logger)
	}
}

func CheckProfileDir(cCxt *cli.Context) (err error) {
	_, err = os.Stat(cCxt.String("profile"))
	if err != nil {
		return fmt.Errorf("Error getting profile dir: %s", err)
	}
	return nil
}

func NewGenerateHandler(logger *logrus.Logger) cli.ActionFunc {
	ll := logger.WithFields(logrus.Fields{
		"component": "GenerateHandler",
	})
	return WrapCliHandler(generateHandler, ll)
}

func generateHandler(cCtx *cli.Context, ll *logrus.Entry) (err error) {
	profilePath := cCtx.String("out")
	if _, err := os.Stat(profilePath); (err != nil) && (os.IsNotExist(err)) {
		if err := os.Mkdir(profilePath, 0700); err != nil {
			return fmt.Errorf("error when create profile directory: %s", err)
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("error when generate key pair: %s", err)
	}
	publicKey := &privateKey.PublicKey

	var privateKeyBytes []byte = x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	var privatePem *os.File
	if privatePem, err = fs.CreateIfNotExist(filepath.Join(profilePath, "private.pem")); err != nil {
		return fmt.Errorf("error when create private.pem: %s \n", err)
	}
	defer privatePem.Close()

	err = pem.Encode(privatePem, privateKeyBlock)
	if err != nil {
		return fmt.Errorf("error when encode private pem: %s \n", err)
	}

	publicKeyBytes := x509.MarshalPKCS1PublicKey(publicKey)
	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	var publicPem *os.File
	if publicPem, err = fs.CreateIfNotExist(filepath.Join(profilePath, "public.pem")); err != nil {
		return fmt.Errorf("error when create public.pem: %s \n", err)
	}
	defer publicPem.Close()

	err = pem.Encode(publicPem, publicKeyBlock)
	if err != nil {
		return fmt.Errorf("error when encode public pem: %s \n", err)
	}
	ll.Println("Generated new key pair")
	return nil
}

func NewAddressHandler(logger *logrus.Logger) cli.ActionFunc {
	ll := logger.WithFields(logrus.Fields{
		"component": "AddressHandler",
	})
	return WrapCliHandler(addressHandler, ll)
}

func addressHandler(cCtx *cli.Context, ll *logrus.Entry) (err error) {
	profilePath := cCtx.String("profile")
	var senderProfile profile.Profile
	if senderProfile, err = profile.ReadProfile(profilePath); err != nil {
		return err
	}
	ll.Debugf("Read sender profile %s\n", profilePath)

	fmt.Printf("Your address is:\n%s\n", senderProfile.GetAddress())

	return nil
}

func NewOnlineHandler(logger *logrus.Logger) cli.ActionFunc {
	ll := logger.WithFields(logrus.Fields{
		"component": "OnlineHandler",
	})
	return WrapCliHandler(WrapCliDaemonHandler(onlineHandler), ll)
}

func onlineHandler(cCtx *cli.Context, ll *logrus.Entry, daemonClient api.DaemonClient) (err error) {
	profilePath := cCtx.String("profile")
	var senderProfile profile.Profile
	if senderProfile, err = profile.ReadProfile(profilePath); err != nil {
		return err
	}

	natsUrl := cCtx.String("nats-url")
	_, err = daemonClient.Online(cCtx.Context, &api.OnlineRequest{
		NatsUrl:       natsUrl,
		SenderAddress: senderProfile.GetAddress(),
	})

	if err != nil {
		return fmt.Errorf("error going online: %s", err)
	}
	return nil
}

func NewOfflineHandler(logger *logrus.Logger) cli.ActionFunc {
	ll := logger.WithFields(logrus.Fields{
		"component": "OfflineHandler",
	})
	return WrapCliHandler(WrapCliDaemonHandler(offlineHandler), ll)
}

func offlineHandler(cCtx *cli.Context, ll *logrus.Entry, daemonClient api.DaemonClient) (err error) {
	if _, err = daemonClient.Offline(context.Background(), &emptypb.Empty{}); err != nil {
		return fmt.Errorf("error going offline: %s", err)
	}
	return nil
}

func NewCreateChatHandler(logger *logrus.Logger) cli.ActionFunc {
	ll := logger.WithFields(logrus.Fields{
		"component": "ChatHandler",
	})
	return WrapCliHandler(WrapCliDaemonHandler(createChatHandler), ll)
}

func createChatHandler(cCtx *cli.Context, ll *logrus.Entry, daemonClient api.DaemonClient) (err error) {
	recepientAddress := cCtx.String("recepient")

	_, err = daemonClient.CreateChat(cCtx.Context, &api.ChatRequest{
		RecepientAddress: recepientAddress,
	})
	return err
}

func NewRmChatHandler(logger *logrus.Logger) cli.ActionFunc {
	ll := logger.WithFields(logrus.Fields{
		"component": "ChatHandler",
	})
	return WrapCliHandler(WrapCliDaemonHandler(rmChatHandler), ll)
}

func rmChatHandler(cCtx *cli.Context, ll *logrus.Entry, daemonClient api.DaemonClient) (err error) {
	recepientAddress := cCtx.String("recepient")

	_, err = daemonClient.DeleteChat(cCtx.Context, &api.ChatRequest{
		RecepientAddress: recepientAddress,
	})
	return err
}

func NewOpenChatHandler(logger *logrus.Logger) cli.ActionFunc {
	ll := logger.WithFields(logrus.Fields{
		"component": "ChatHandler",
	})
	return WrapCliHandler(WrapCliDaemonHandler(openChatHandler), ll)
}

func openChatHandler(cCtx *cli.Context, ll *logrus.Entry, daemonClient api.DaemonClient) (err error) {
	var daemonSendClient api.Daemon_SendClient
	if daemonSendClient, err = daemonClient.Send(context.Background()); err != nil {
		return fmt.Errorf("failed send: %s", err)
	}

	g := errgroup.Group{}
	g.Go(func() (err error) {
		var cmsg *api.ChatMessage
		for {
			cmsg, err = daemonSendClient.Recv()
			if err != nil {
				if e, ok := status.FromError(err); ok && (e.Code() == codes.Canceled) {
					ll.Debugf("Exiting cli recv loop: %s", err)
					return nil
				} else {
					return fmt.Errorf("Unexpected error from stream: %s", err)
				}
			}

			fmt.Printf("%s %s\n", cmsg.Time.AsTime(), cmsg.Text)
		}
	})

	g.Go(func() (err error) {
		defer daemonSendClient.CloseSend()
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			cmsg := &api.ChatMessage{
				Text: scanner.Text(),
				Time: timestamppb.Now(),
			}
			if err = daemonSendClient.Send(cmsg); err != nil {
				return fmt.Errorf("Unexpected error sending message: %s", err)
			}
			ll.Debugf("Sent message: %s", cmsg)
		}
		if scanner.Err() != nil {
			return fmt.Errorf("Got error from scanner: %s", err)
		}
		ll.Debugln("Got EOF, exiting cli send loop")
		return nil
	})

	if err = g.Wait(); err != nil {
		return err
	}
	return nil
}
