package handlers

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aaletov/nats-chat/pkg/profiles"
	"github.com/aaletov/nats-chat/pkg/types"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func NewGenerateHandler(logger *logrus.Logger) cli.ActionFunc {
	return func(cCtx *cli.Context) error {
		ll := logger.WithFields(logrus.Fields{
			"component": "GenerateHandler",
		})

		outPath := cCtx.String("out")
		if _, err := os.Stat(outPath); (err != nil) && (os.IsNotExist(err)) {
			return fmt.Errorf("directory does not exist: %s", err)
		}

		profilePath := outPath
		if !cCtx.IsSet("out") {
			profilePath = filepath.Join(outPath, ".natschat")
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
		if privatePem, err = createIfNotExist(filepath.Join(profilePath, "private.pem")); err != nil {
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
		if publicPem, err = createIfNotExist(filepath.Join(profilePath, "public.pem")); err != nil {
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
}

func NewRunHandler(logger *logrus.Logger) cli.ActionFunc {
	return func(cCtx *cli.Context) error {
		ll := logger.WithFields(logrus.Fields{
			"component": "RunHandler",
		})

		profilePath := cCtx.String("profile")
		recepientKeyPath := cCtx.String("recepient-key")

		var (
			err              error
			senderProfile    profiles.SenderProfile
			recepientProfile profiles.RecepientProfile
		)

		if senderProfile, err = readSenderProfile(profilePath); err != nil {
			return err
		}
		ll.Println("Read sender profile")

		if recepientProfile, err = readRecepientProfile(recepientKeyPath); err != nil {
			return err
		}
		ll.Println("Read recepient profile")

		options := []nats.Option{nats.Timeout(10 * time.Second)}
		var nc *nats.Conn
		if nc, err = nats.Connect(cCtx.String("nats-url"), options...); err != nil {
			return fmt.Errorf("error connecting to nats instance: %s", err)
		}
		defer nc.Close()
		ll.Println("Connected to the nats server")

		session := NewSession(logger, nc, senderProfile)
		session.Open()
		defer session.Close()
		var conn *ChatConnection
		if conn, err = session.Dial(recepientProfile.GetAddress()); err != nil {
			return fmt.Errorf("error dialing %s: %s", recepientProfile.GetAddress(), err)
		}
		ll.Printf("Successfully dialed: %s\n", recepientProfile.GetAddress())
		defer conn.Close()

		go func() {
			for cmsg := range conn.IncomingChan {
				fmt.Printf("%s.%s\n", cmsg.Time, cmsg.Text)
			}
		}()

		scanner := bufio.NewScanner(os.Stdin)
		func() {
			for {
				select {
				case isOnline := <-conn.OnlineChan:
					if isOnline {
						panic("not handled")
					} else {
						close(conn.OutcomingChan)
						fmt.Println("User went offline. Press ENTER to exit")
						return
					}
				default:
					if !scanner.Scan() {
						if scanner.Err() != nil {
							panic("not handled")
						}
						close(conn.OutcomingChan)
						return
					}
					text := scanner.Text()
					cmsg := types.ChatMessage{
						Time: time.Now(),
						Text: text,
					}
					conn.OutcomingChan <- cmsg
				}
			}
		}()

		return nil
	}
}

func readPrivateKey(privateKeyPath string) (*rsa.PrivateKey, error) {
	var err error
	if _, err := os.Stat(privateKeyPath); (err != nil) && (os.IsNotExist(err)) {
		return nil, fmt.Errorf("file does not exist: %s", err)
	}
	var privatePemFile *os.File
	if privatePemFile, err = os.Open(privateKeyPath); err != nil {
		return nil, fmt.Errorf("error opening private key: %s", err)
	}
	defer privatePemFile.Close()
	var privatePemBytes []byte
	if privatePemBytes, err = io.ReadAll(privatePemFile); err != nil {
		return nil, fmt.Errorf("error reading private key: %s", err)
	}
	privateKeyBlock, _ := pem.Decode(privatePemBytes)
	var privateKey *rsa.PrivateKey
	if privateKey, err = x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes); err != nil {
		return nil, fmt.Errorf("error parsing private key: %s", err)
	}
	return privateKey, nil
}

func readRecepientKey(recepientKeyPath string) (*rsa.PublicKey, error) {
	var err error
	if _, err = os.Stat(recepientKeyPath); (err != nil) && (os.IsNotExist(err)) {
		return nil, fmt.Errorf("file does not exist: %s", err)
	}

	var recepientPemFile *os.File
	if recepientPemFile, err = os.Open(recepientKeyPath); err != nil {
		return nil, fmt.Errorf("error opening recepient key: %s", err)
	}
	defer recepientPemFile.Close()
	var recepientPemBytes []byte
	if recepientPemBytes, err = io.ReadAll(recepientPemFile); err != nil {
		return nil, fmt.Errorf("error reading recepient key: %s", err)
	}
	recepientKeyBlock, _ := pem.Decode(recepientPemBytes)
	var recepientKey *rsa.PublicKey
	if recepientKey, err = x509.ParsePKCS1PublicKey(recepientKeyBlock.Bytes); err != nil {
		return nil, fmt.Errorf("error parsing recepient key: %s", err)
	}
	return recepientKey, nil
}

func readSenderProfile(profilePath string) (profiles.SenderProfile, error) {
	var err error
	if _, err := os.Stat(profilePath); (err != nil) && (os.IsNotExist(err)) {
		return profiles.SenderProfile{}, fmt.Errorf("directory does not exist: %s", err)
	}
	privateKeyPath := filepath.Join(profilePath, "private.pem")
	var privateKey *rsa.PrivateKey
	if privateKey, err = readPrivateKey(privateKeyPath); err != nil {
		return profiles.SenderProfile{}, fmt.Errorf("error reading private key: %s", err)
	}

	var profile profiles.SenderProfile
	if profile, err = profiles.NewSenderProfile(privateKey); err != nil {
		return profiles.SenderProfile{}, fmt.Errorf("error constructing sender profile: %s", err)
	}
	return profile, nil
}

func readRecepientProfile(recepientKeyPath string) (profiles.RecepientProfile, error) {
	var (
		recepientKey *rsa.PublicKey
		err          error
	)
	if recepientKey, err = readRecepientKey(recepientKeyPath); err != nil {
		return profiles.RecepientProfile{}, fmt.Errorf("error reading recepient key: %s", err)
	}

	var profile profiles.RecepientProfile
	if profile, err = profiles.NewRecepientProfile(recepientKey); err != nil {
		return profiles.RecepientProfile{}, fmt.Errorf("error constructing recepient profile")
	}

	return profile, nil
}

func createIfNotExist(filepath string) (*os.File, error) {
	var (
		err  error
		file *os.File
	)

	if _, err = os.Stat(filepath); err == nil {
		return nil, errors.New("file exists")
	}

	if file, err = os.Create(filepath); err != nil {
		return nil, err
	}

	return file, nil
}
