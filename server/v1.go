package server

import (
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"net"

	v1 "github.com/csnewman/dyndirect/server/internal/v1"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var errRequestMissingInCtx = errors.New("request missing in ctx")

type v1API struct {
	tokenHash []byte
	store     Store
}

func (v *v1API) GetOverview(
	ctx context.Context,
	_ v1.GetOverviewRequestObject,
) (v1.GetOverviewResponseObject, error) {
	r, ok := requestFromCtx(ctx)
	if !ok {
		return nil, errRequestMissingInCtx
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, err
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		return nil, err
	}

	return v1.GetOverview200JSONResponse{
		Version:  Version,
		ClientIp: userIP.String(),
	}, nil
}

func (v *v1API) GenerateSubdomain(
	_ context.Context,
	_ v1.GenerateSubdomainRequestObject,
) (v1.GenerateSubdomainResponseObject, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	token := v.generateToken(id)

	return v1.GenerateSubdomain200JSONResponse{
		Id:    id,
		Token: token,
	}, nil
}

func (v *v1API) SubdomainAcmeChallenge(
	ctx context.Context,
	r v1.SubdomainAcmeChallengeRequestObject,
) (v1.SubdomainAcmeChallengeResponseObject, error) {
	expectedToken := v.generateToken(r.SubdomainId)

	if subtle.ConstantTimeCompare([]byte(expectedToken), []byte(r.Body.Token)) != 1 {
		return v1.SubdomainAcmeChallenge403JSONResponse{
			Error:   "invalid-token",
			Message: "The provided token is not valid for the subdomain.",
		}, nil
	}

	if err := v.store.SetACMEChallengeTokens(ctx, r.SubdomainId, r.Body.Values); err != nil {
		return nil, err
	}

	return v1.SubdomainAcmeChallenge200Response{}, nil
}

func (v *v1API) generateToken(id uuid.UUID) string {
	buf := v.tokenHash
	buf = append(buf, id[:]...)
	hash := sha512.Sum512(buf)

	return hex.EncodeToString(hash[:])
}
