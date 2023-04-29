package server

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	v1 "github.com/csnewman/dyndirect/server/internal/v1"
	"github.com/google/uuid"
)

type v1API struct {
	tokenHash []byte
}

func (v *v1API) Overview(
	_ context.Context,
	_ v1.OverviewRequestObject,
) (v1.OverviewResponseObject, error) {
	return v1.Overview200JSONResponse{
		Version: Version,
	}, nil
}

func (v *v1API) NewSubdomain(
	_ context.Context,
	_ v1.NewSubdomainRequestObject,
) (v1.NewSubdomainResponseObject, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	token := v.generateToken(id)

	return v1.NewSubdomain200JSONResponse{
		Id:    id,
		Token: token,
	}, nil
}

func (v *v1API) SubdomainAcmeChallenge(
	_ context.Context,
	_ v1.SubdomainAcmeChallengeRequestObject,
) (v1.SubdomainAcmeChallengeResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (v *v1API) generateToken(id uuid.UUID) string {
	buf := append(v.tokenHash[:], id[:]...)
	hash := sha512.Sum512(buf)
	return hex.EncodeToString(hash[:])
}
