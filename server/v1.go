package server

import (
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	v1 "github.com/csnewman/dyndirect/server/internal/v1"
	"github.com/google/uuid"
)

type v1API struct {
	tokenHash []byte
	store     Store
}

func (v *v1API) GetOverview(
	_ context.Context,
	_ v1.GetOverviewRequestObject,
) (v1.GetOverviewResponseObject, error) {
	return v1.GetOverview200JSONResponse{
		Version: Version,
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
	buf := append(v.tokenHash[:], id[:]...)
	hash := sha512.Sum512(buf)
	return hex.EncodeToString(hash[:])
}
