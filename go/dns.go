package dsdm

import (
	"context"

	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/google/uuid"
)

type DNSChallengeProvider struct {
	//nolint:containedctx
	ctx    context.Context
	client *Client
	id     uuid.UUID
	token  string
}

func (c *Client) NewDNSChallengeProvider(
	ctx context.Context,
	id uuid.UUID,
	token string,
) *DNSChallengeProvider {
	return &DNSChallengeProvider{
		ctx:    ctx,
		client: c,
		id:     id,
		token:  token,
	}
}

func (p *DNSChallengeProvider) Present(domain string, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)

	return p.client.SetSubdomainACMEChallenge(p.ctx, SubdomainACMEChallengeRequest{
		ID:    p.id,
		Token: p.token,
		Values: []string{
			info.Value,
		},
	})
}

func (p *DNSChallengeProvider) CleanUp(_ string, _ string, _ string) error {
	return nil
}
