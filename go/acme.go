package dsdm

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	llog "github.com/go-acme/lego/v4/log"
	"github.com/go-acme/lego/v4/registration"
	"github.com/google/uuid"
)

var (
	ErrAccountCreationError = errors.New("account creation failed")
	ErrUnsupportedProvider  = errors.New("unsupported provider")
)

type acmeUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *acmeUser) GetEmail() string {
	return u.Email
}

func (u *acmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *acmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

type AcquireCertificateRequest struct {
	ID         uuid.UUID
	Domain     string
	Token      string
	Provider   string
	KeyType    certcrypto.KeyType
	Timeout    time.Duration
	SilenceLog bool
}

type CertificateResponse struct {
	Domain            string
	CertURL           string
	CertStableURL     string
	PrivateKey        []byte
	Certificate       []byte
	IssuerCertificate []byte
	CSR               []byte
}

func (c *Client) AcquireCertificate(
	ctx context.Context,
	request AcquireCertificateRequest,
) (*CertificateResponse, error) {
	if request.SilenceLog {
		llog.Logger = &nullLogger{}
	}

	ctx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	emailID1, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	emailID2, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	// Email is unlikely to exist
	email := fmt.Sprintf("%s@%s.com", emailID1, emailID2)

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	user := &acmeUser{
		Email: email,
		key:   privateKey,
	}

	config := lego.NewConfig(user)
	config.Certificate.KeyType = request.KeyType

	registerOpts := registration.RegisterEABOptions{
		TermsOfServiceAgreed: true,
		Kid:                  "",
		HmacEncoded:          "",
	}

	switch request.Provider {
	case ProviderZeroSSL:
		config.CADirURL = zeroSSLURL

		account, err := generateZeroSslAccount(ctx, email, request.Timeout)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrAccountCreationError, err)
		}

		if !account.Success {
			return nil, fmt.Errorf("%w: unknown failure", ErrAccountCreationError)
		}

		registerOpts.Kid = account.EABKID
		registerOpts.HmacEncoded = account.EABHMACKey
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, request.Provider)
	}

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	dnsChallenge := c.NewDNSChallengeProvider(ctx, request.ID, request.Token)

	// Disable propagation check, does not work currently
	err = client.Challenge.SetDNS01Provider(dnsChallenge, dns01.DisableCompletePropagationRequirement())
	if err != nil {
		return nil, err
	}

	reg, err := client.Registration.RegisterWithExternalAccountBinding(registerOpts)
	if err != nil {
		return nil, err
	}

	user.Registration = reg

	response, err := client.Certificate.Obtain(certificate.ObtainRequest{
		Domains:                        []string{fmt.Sprintf("*.%s", request.Domain)},
		Bundle:                         true,
		AlwaysDeactivateAuthorizations: true,
	})
	if err != nil {
		return nil, err
	}

	return &CertificateResponse{
		Domain:            response.Domain,
		CertURL:           response.CertURL,
		CertStableURL:     response.CertStableURL,
		PrivateKey:        response.PrivateKey,
		Certificate:       response.Certificate,
		IssuerCertificate: response.IssuerCertificate,
		CSR:               response.CSR,
	}, nil
}

type nullLogger struct{}

func (n nullLogger) Fatal(_ ...interface{}) {
}

func (n nullLogger) Fatalln(_ ...interface{}) {
}

func (n nullLogger) Fatalf(_ string, _ ...interface{}) {
}

func (n nullLogger) Print(_ ...interface{}) {
}

func (n nullLogger) Println(_ ...interface{}) {
}

func (n nullLogger) Printf(_ string, _ ...interface{}) {
}
