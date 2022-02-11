package searchV2

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type DashboardQuery struct {
	Query string
}

type SearchService interface {
	DoDashboardQuery(ctx context.Context, user *backend.User, query DashboardQuery) *backend.DataResponse
}
