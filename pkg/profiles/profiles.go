package profiles

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
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
	builder := &strings.Builder{}
	_, err := base64.NewEncoder(base64.StdEncoding, builder).Write(publicKeyPem)
	if err != nil {
		return "", fmt.Errorf("error encoding key in base64: %s", err)
	}

	return builder.String(), nil
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
