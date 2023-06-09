package dsdm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/csnewman/dyndirect/go/internal"
	"github.com/google/uuid"
)

const DynDirect string = "https://v1.dyn.direct/"

type OverviewResponse = internal.OverviewResponse

type SubdomainResponse = internal.NewSubdomainResponse

type SubdomainACMEChallengeRequest struct {
	ID     uuid.UUID
	Token  string
	Values []string
}

type Client struct {
	server string
	v1     *internal.Client
}

func New(server string) (*Client, error) {
	c, err := internal.NewClient(server)
	if err != nil {
		return nil, err
	}

	return &Client{
		server: server,
		v1:     c,
	}, nil
}

func (c *Client) GetOverview(ctx context.Context) (*OverviewResponse, error) {
	resp, err := c.v1.GetOverview(ctx, c.requestHook)
	if err != nil {
		return nil, err
	}

	return parseResponse[OverviewResponse](resp)
}

func (c *Client) RequestSubdomain(ctx context.Context) (*SubdomainResponse, error) {
	resp, err := c.v1.GenerateSubdomain(ctx, c.requestHook)
	if err != nil {
		return nil, err
	}

	return parseResponse[SubdomainResponse](resp)
}

func GetDomainForIP(rootDomain string, ip net.IP) string {
	rootDomain = strings.ToLower(rootDomain)

	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%s-v4.%s", strings.ReplaceAll(v4.String(), ".", "-"), rootDomain)
	}

	return fmt.Sprintf("%s-v6.%s", strings.ReplaceAll(ip.String(), ":", "-"), rootDomain)
}

func (c *Client) SetSubdomainACMEChallenge(
	ctx context.Context,
	req SubdomainACMEChallengeRequest,
) error {
	resp, err := c.v1.SubdomainAcmeChallenge(ctx, req.ID, internal.SubdomainAcmeChallengeRequest{
		Token:  req.Token,
		Values: req.Values,
	}, c.requestHook)
	if err != nil {
		return err
	}

	return parseEmptyResponse(resp)
}

func (c *Client) requestHook(_ context.Context, req *http.Request) error {
	req.Header.Set("User-Agent", "dsdm-go-client/1.0")

	return nil
}

type APIError struct {
	Status    int
	ErrorCode string
	Message   string
}

func (e APIError) Error() string {
	return fmt.Sprintf("dsdm: api error: %d %s '%s'", e.Status, e.ErrorCode, e.Message)
}

func parseResponse[T any](rsp *http.Response) (*T, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)

	defer rsp.Body.Close()

	if err != nil {
		return nil, err
	}

	ct := rsp.Header.Get("Content-Type")
	if !strings.Contains(ct, "json") {
		return nil, APIError{
			Status:    rsp.StatusCode,
			ErrorCode: "invalid-response",
			Message:   fmt.Sprintf("Unexpected content-type %s", ct),
		}
	}

	if rsp.StatusCode == http.StatusOK {
		var dest T
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}

		return &dest, nil
	}

	var dest internal.ErrorResponse
	if err := json.Unmarshal(bodyBytes, &dest); err != nil {
		return nil, err
	}

	return nil, APIError{
		Status:    rsp.StatusCode,
		ErrorCode: dest.Error,
		Message:   dest.Message,
	}
}

func parseEmptyResponse(rsp *http.Response) error {
	bodyBytes, err := io.ReadAll(rsp.Body)

	defer rsp.Body.Close()

	if err != nil {
		return err
	}

	if rsp.StatusCode == http.StatusOK {
		return nil
	}

	ct := rsp.Header.Get("Content-Type")
	if !strings.Contains(ct, "json") {
		return APIError{
			Status:    rsp.StatusCode,
			ErrorCode: "invalid-response",
			Message:   fmt.Sprintf("Unexpected content-type %s", ct),
		}
	}

	var dest internal.ErrorResponse
	if err := json.Unmarshal(bodyBytes, &dest); err != nil {
		return err
	}

	return APIError{
		Status:    rsp.StatusCode,
		ErrorCode: dest.Error,
		Message:   dest.Message,
	}
}
