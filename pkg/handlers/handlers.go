package handlers

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aaletov/nats-chat/pkg/chatmsg"
	"github.com/aaletov/nats-chat/pkg/profiles"
	"github.com/nats-io/nats.go"
	"github.com/urfave/cli/v2"
)

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

func GenerateHandler(cCtx *cli.Context) error {
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

	return nil
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
	if privatePemBytes, err = ioutil.ReadAll(privatePemFile); err != nil {
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
	if recepientPemBytes, err = ioutil.ReadAll(recepientPemFile); err != nil {
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

func RunHandler(cCtx *cli.Context) error {
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

	if recepientProfile, err = readRecepientProfile(recepientKeyPath); err != nil {
		return err
	}

	var nc *nats.Conn
	if nc, err = nats.Connect(cCtx.String("nats-url")); err != nil {
		return fmt.Errorf("error connecting to nats instance: %s", err)
	}
	defer nc.Close()

	nc.Subscribe(recepientProfile.GetAddress(), func(msg *nats.Msg) {
		var cmsg chatmsg.ChatMessage
		if err := json.Unmarshal(msg.Data, &cmsg); err != nil {
			msg.Nak()
			return
		}
		fmt.Printf("%s %s\n", cmsg.Time.String(), cmsg.Text)
		msg.Ack()
	})

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(os.Stdin)
		for {
			scanner.Scan()
			text := scanner.Text()
			cmsg := chatmsg.ChatMessage{
				Time: time.Now(),
				Text: text,
			}
			var msgBytes []byte
			if msgBytes, err = json.Marshal(cmsg); err != nil {
				log.Printf("error marshalling message: %s", err)
			}
			nc.Publish(senderProfile.GetAddress(), msgBytes)
		}
	}()
	wg.Wait()

	return nil
}
