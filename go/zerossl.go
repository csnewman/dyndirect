package dsdm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	zeroSSLAccountEndpoint = "https://api.zerossl.com/acme/eab-credentials-email"
	zeroSSLURL             = "https://acme.zerossl.com/v2/DV90"
	ProviderZeroSSL        = "zerossl"
)

type zeroSSLAccountResp struct {
	Success bool `json:"success"`

	//nolint:tagliatelle
	EABKID string `json:"eab_kid"`

	//nolint:tagliatelle
	EABHMACKey string `json:"eab_hmac_key"`
}

func generateZeroSslAccount(ctx context.Context, email string, timeout time.Duration) (*zeroSSLAccountResp, error) {
	form := url.Values{"email": {email}}
	formReader := bytes.NewReader([]byte(form.Encode()))

	hc := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, zeroSSLAccountEndpoint, formReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("%w: unexpected response: %s", ErrAccountCreationError, string(body))
	}

	var decoded zeroSSLAccountResp

	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	return &decoded, nil
}
