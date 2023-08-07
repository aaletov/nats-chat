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
	"sync"

	api "github.com/aaletov/nats-chat/api/generated"
	"github.com/aaletov/nats-chat/pkg/fs"
	"github.com/aaletov/nats-chat/pkg/profile"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Connector struct {
	Client api.DaemonClient
	Conn   *grpc.ClientConn
}

func NewConnector(cCtx *cli.Context) (Connector, error) {
	var (
		homeDir string
		err     error
	)

	if homeDir, err = os.UserHomeDir(); err != nil {
		return Connector{}, fmt.Errorf("Unable to get user's home directory: %s", err)
	}
	PROTOCOL := "unix"
	SOCKET := filepath.Join(homeDir, ".natschat/socket/natschat.sock")
	dialOption := grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return net.Dial(PROTOCOL, s)
	})
	secOption := grpc.WithTransportCredentials(insecure.NewCredentials())

	conn, err := grpc.Dial(SOCKET, dialOption, secOption)
	if err != nil {
		return Connector{}, err
	}
	daemonClient := api.NewDaemonClient(conn)

	return Connector{Client: daemonClient, Conn: conn}, nil
}

func (c *Connector) Close() error {
	if c.Conn == nil {
		return nil
	}

	return c.Conn.Close()
}

type natscli func(*cli.Context, *logrus.Logger) error

func Wrapnatscli(handler natscli, logger *logrus.Logger) cli.ActionFunc {
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
	return Wrapnatscli(generateHandler, logger)
}

func generateHandler(cCtx *cli.Context, logger *logrus.Logger) (err error) {
	ll := logger.WithFields(logrus.Fields{
		"component": "GenerateHandler",
	})

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
	return Wrapnatscli(addressHandler, logger)
}

func addressHandler(cCtx *cli.Context, logger *logrus.Logger) (err error) {
	ll := logger.WithFields(logrus.Fields{
		"component": "AddressHandler",
	})

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
	return Wrapnatscli(onlineHandler, logger)
}

func onlineHandler(cCtx *cli.Context, logger *logrus.Logger) (err error) {
	ll := logger.WithFields(logrus.Fields{
		"component": "OnlineHandler",
	})

	var connector Connector
	if connector, err = NewConnector(cCtx); err != nil {
		return fmt.Errorf("failed to initialize connector: %s", err)
	}
	defer connector.Close()
	ll.Println("Initialized connector")

	natsUrl := cCtx.String("nats-url")
	profilePath := cCtx.String("profile")
	if !cCtx.IsSet("profile") {
		var homeDir string
		if homeDir, err = os.UserHomeDir(); err != nil {
			return err
		}
		profilePath = filepath.Join(homeDir, ".natschat")
	}
	var senderProfile profile.Profile
	if senderProfile, err = profile.ReadProfile(profilePath); err != nil {
		return err
	}

	_, err = connector.Client.Online(cCtx.Context, &api.OnlineRequest{
		NatsUrl:       natsUrl,
		SenderAddress: senderProfile.GetAddress(),
	})

	if err != nil {
		return fmt.Errorf("error going online: %s", err)
	}
	return nil
}

func NewOfflineHandler(logger *logrus.Logger) cli.ActionFunc {
	return Wrapnatscli(offlineHandler, logger)
}

func offlineHandler(cCtx *cli.Context, logger *logrus.Logger) (err error) {
	ll := logger.WithFields(logrus.Fields{
		"component": "OfflineHandler",
	})

	var connector Connector
	if connector, err = NewConnector(cCtx); err != nil {
		return fmt.Errorf("failed to initialize connector: %s", err)
	}
	defer connector.Close()
	ll.Println("Initialized connector")

	if _, err = connector.Client.Offline(context.Background(), &emptypb.Empty{}); err != nil {
		return fmt.Errorf("error going offline: %s", err)
	}
	return nil
}

func NewCreateChatHandler(logger *logrus.Logger) cli.ActionFunc {
	return Wrapnatscli(createChatHandler, logger)
}

func createChatHandler(cCtx *cli.Context, logger *logrus.Logger) (err error) {
	ll := logger.WithFields(logrus.Fields{
		"component": "ChatHandler",
	})

	var connector Connector
	if connector, err = NewConnector(cCtx); err != nil {
		return fmt.Errorf("failed to initialize connector: %s", err)
	}
	defer connector.Close()
	ll.Println("Initialized connector")

	recepientAddress := cCtx.String("recepient")

	_, err = connector.Client.CreateChat(cCtx.Context, &api.ChatRequest{
		RecepientAddress: recepientAddress,
	})
	return err
}

func NewRmChatHandler(logger *logrus.Logger) cli.ActionFunc {
	return Wrapnatscli(rmChatHandler, logger)
}

func rmChatHandler(cCtx *cli.Context, logger *logrus.Logger) (err error) {
	ll := logger.WithFields(logrus.Fields{
		"component": "ChatHandler",
	})

	var connector Connector
	if connector, err = NewConnector(cCtx); err != nil {
		return fmt.Errorf("failed to initialize connector: %s", err)
	}
	defer connector.Close()
	ll.Println("Initialized connector")

	recepientAddress := cCtx.String("recepient")

	_, err = connector.Client.DeleteChat(cCtx.Context, &api.ChatRequest{
		RecepientAddress: recepientAddress,
	})
	return err
}

func NewOpenChatHandler(logger *logrus.Logger) cli.ActionFunc {
	return Wrapnatscli(openChatHandler, logger)
}

func openChatHandler(cCtx *cli.Context, logger *logrus.Logger) (err error) {
	ll := logger.WithFields(logrus.Fields{
		"component": "ChatHandler",
	})

	var connector Connector
	if connector, err = NewConnector(cCtx); err != nil {
		return fmt.Errorf("failed to initialize connector: %s", err)
	}
	defer connector.Close()
	ll.Println("Initialized connector")

	var daemonSendClient api.Daemon_SendClient
	if daemonSendClient, err = connector.Client.Send(context.Background()); err != nil {
		return fmt.Errorf("failed send: %s", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		var (
			err  error
			cmsg *api.ChatMessage
		)
		for {
			if cmsg, err = daemonSendClient.Recv(); err != nil {
				ll.Fatalf("Got error from stream: %s", err)
			}
			fmt.Printf("%s %s\n", cmsg.Time, cmsg.Text)
		}
		ll.Debugln("Exiting cli recv loop")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			cmsg := &api.ChatMessage{
				Text: scanner.Text(),
				Time: timestamppb.Now(),
			}
			daemonSendClient.Send(cmsg)
			ll.Debugf("Sent message: %s", cmsg)
		}
		if scanner.Err() != nil {
			ll.Fatalf("Got error from scanner: %s", err)
		}
		ll.Debugln("Exiting cli send loop")
	}()
	wg.Wait()
	return nil
}
