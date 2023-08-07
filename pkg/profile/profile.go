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

type Profile struct {
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

func NewProfile(privateKey *rsa.PrivateKey) (Profile, error) {
	var (
		err       error
		publicKey = &privateKey.PublicKey
		address   string
	)

	if address, err = getAddress(publicKey); err != nil {
		return Profile{}, fmt.Errorf("error getting address of sender: %s", err)
	}

	return Profile{
		privateKey: privateKey,
		publicKey:  publicKey,
		address:    address,
	}, nil
}

func (s Profile) GetPrivateKey() *rsa.PrivateKey {
	return s.privateKey
}

func (s Profile) GetPublicKey() *rsa.PublicKey {
	return s.publicKey
}

func (s Profile) GetAddress() string {
	return s.address
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

func ReadProfile(profilePath string) (Profile, error) {
	var err error
	if _, err := os.Stat(profilePath); (err != nil) && (os.IsNotExist(err)) {
		return Profile{}, fmt.Errorf("directory does not exist: %s", err)
	}
	privateKeyPath := filepath.Join(profilePath, "private.pem")
	var privateKey *rsa.PrivateKey
	if privateKey, err = ReadPrivateKey(privateKeyPath); err != nil {
		return Profile{}, fmt.Errorf("error reading private key: %s", err)
	}

	var profile Profile
	if profile, err = NewProfile(privateKey); err != nil {
		return Profile{}, fmt.Errorf("error constructing sender profile: %s", err)
	}
	return profile, nil
}
