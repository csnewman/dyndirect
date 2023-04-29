package server

import (
	"context"
	v1 "github.com/csnewman/dyndirect/server/internal/v1"
)

type v1API struct {
}

func (v v1API) Overview(
	_ context.Context,
	_ v1.OverviewRequestObject,
) (v1.OverviewResponseObject, error) {
	return v1.Overview200JSONResponse{
		Version: Version,
	}, nil
}

func (v v1API) NewSubdomain(_ context.Context, _ v1.NewSubdomainRequestObject) (v1.NewSubdomainResponseObject, error) {
	//TODO implement me
	panic("implement me")
}

func (v v1API) SubdomainAcmeChallenge(
	_ context.Context,
	_ v1.SubdomainAcmeChallengeRequestObject,
) (v1.SubdomainAcmeChallengeResponseObject, error) {
	//TODO implement me
	panic("implement me")
}
