package profile

import (
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcutil/base58"
)

type SenderProfile struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	address    string
}

func getAddress(publicKey *rsa.PublicKey) (string, error) {
	publicKeyBytes := x509.MarshalPKCS1PublicKey(publicKey)
	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	publicKeyPem := pem.EncodeToMemory(publicKeyBlock)
	hash256 := sha256.Sum256(publicKeyPem)
	hash128 := md5.Sum(hash256[:])
	address := base58.Encode(hash128[:])

	return address, nil
}

func NewSenderProfile(privateKey *rsa.PrivateKey) (SenderProfile, error) {
	var (
		err       error
		publicKey = &privateKey.PublicKey
		address   string
	)

	if address, err = getAddress(publicKey); err != nil {
		return SenderProfile{}, fmt.Errorf("error getting address of sender: %s", err)
	}

	return SenderProfile{
		privateKey: privateKey,
		publicKey:  publicKey,
		address:    address,
	}, nil
}

func (s SenderProfile) GetPrivateKey() *rsa.PrivateKey {
	return s.privateKey
}

func (s SenderProfile) GetPublicKey() *rsa.PublicKey {
	return s.publicKey
}

func (s SenderProfile) GetAddress() string {
	return s.address
}

type RecepientProfile struct {
	publicKey *rsa.PublicKey
	address   string
}

func NewRecepientProfile(publicKey *rsa.PublicKey) (RecepientProfile, error) {
	var (
		err     error
		address string
	)

	if address, err = getAddress(publicKey); err != nil {
		return RecepientProfile{}, fmt.Errorf("error getting address of recepient: %s", err)
	}

	return RecepientProfile{
		publicKey: publicKey,
		address:   address,
	}, nil
}

func (r RecepientProfile) GetPublicKey() *rsa.PublicKey {
	return r.publicKey
}

func (r RecepientProfile) GetAddress() string {
	return r.address
}

func ReadPrivateKey(privateKeyPath string) (*rsa.PrivateKey, error) {
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

func ReadRecepientKey(recepientKeyPath string) (*rsa.PublicKey, error) {
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

func ReadSenderProfile(profilePath string) (SenderProfile, error) {
	var err error
	if _, err := os.Stat(profilePath); (err != nil) && (os.IsNotExist(err)) {
		return SenderProfile{}, fmt.Errorf("directory does not exist: %s", err)
	}
	privateKeyPath := filepath.Join(profilePath, "private.pem")
	var privateKey *rsa.PrivateKey
	if privateKey, err = ReadPrivateKey(privateKeyPath); err != nil {
		return SenderProfile{}, fmt.Errorf("error reading private key: %s", err)
	}

	var profile SenderProfile
	if profile, err = NewSenderProfile(privateKey); err != nil {
		return SenderProfile{}, fmt.Errorf("error constructing sender profile: %s", err)
	}
	return profile, nil
}

func ReadRecepientProfile(recepientKeyPath string) (RecepientProfile, error) {
	var (
		recepientKey *rsa.PublicKey
		err          error
	)
	if recepientKey, err = ReadRecepientKey(recepientKeyPath); err != nil {
		return RecepientProfile{}, fmt.Errorf("error reading recepient key: %s", err)
	}

	var profile RecepientProfile
	if profile, err = NewRecepientProfile(recepientKey); err != nil {
		return RecepientProfile{}, fmt.Errorf("error constructing recepient profile")
	}

	return profile, nil
}
