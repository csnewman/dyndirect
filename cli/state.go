package cli

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"time"

	dsdm "github.com/csnewman/dyndirect/go"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/google/uuid"
)

type state struct {
	ID          uuid.UUID `json:"id"`
	Domain      string    `json:"domain"`
	Token       string    `json:"token"`
	Certificate []byte    `json:"cert"`
	PrivateKey  []byte    `json:"cert_private_key"` //nolint:tagliatelle
	IssueDate   time.Time `json:"issue_date"`       //nolint:tagliatelle
}

func getStatePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	dynDir := path.Join(configDir, "dyndirect")
	if err := os.MkdirAll(dynDir, os.ModePerm); err != nil {
		return "", err
	}

	return path.Join(dynDir, "state.json"), nil
}

func getState() (*state, error) {
	filePath, err := getStatePath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return &state{}, nil
	} else if err != nil {
		return nil, err
	}

	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return &state{}, nil
	}

	var state state

	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func setState(s *state) error {
	filePath, err := getStatePath()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	defer file.Close()

	encoder := json.NewEncoder(file)

	return encoder.Encode(s)
}

func GetDomain() (string, error) {
	s, err := getState()
	if err != nil {
		return "", err
	}

	return s.Domain, nil
}

func IssueDomain(ctx context.Context) (string, error) {
	c, err := dsdm.New(dsdm.DynDirect)
	if err != nil {
		return "", err
	}

	resp, err := c.RequestSubdomain(ctx)
	if err != nil {
		return "", err
	}

	return resp.Domain, setState(&state{
		ID:          resp.Id,
		Domain:      resp.Domain,
		Token:       resp.Token,
		Certificate: nil,
		PrivateKey:  nil,
		IssueDate:   time.Time{},
	})
}

func AcquireCertificate(ctx context.Context) error {
	s, err := getState()
	if err != nil {
		return err
	}

	c, err := dsdm.New(dsdm.DynDirect)
	if err != nil {
		return err
	}

	resp, err := c.AcquireCertificate(ctx, dsdm.AcquireCertificateRequest{
		ID:         s.ID,
		Domain:     s.Domain,
		Token:      s.Token,
		Provider:   dsdm.ProviderZeroSSL,
		KeyType:    certcrypto.RSA4096,
		Timeout:    time.Second * 120,
		SilenceLog: true,
	})
	if err != nil {
		return err
	}

	s.Certificate = resp.Certificate
	s.PrivateKey = resp.PrivateKey
	s.IssueDate = time.Now()

	return setState(s)
}

func HasCertificate() (bool, error) {
	s, err := getState()
	if err != nil {
		return false, err
	}

	return len(s.Certificate) > 0, nil
}

func GetCertificate() (tls.Certificate, error) {
	s, err := getState()
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(s.Certificate, s.PrivateKey)
}
