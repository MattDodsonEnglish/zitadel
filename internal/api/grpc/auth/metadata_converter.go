package auth

import (
	"github.com/caos/zitadel/internal/api/grpc/metadata"
	"github.com/caos/zitadel/internal/api/grpc/object"
	"github.com/caos/zitadel/internal/domain"
	"github.com/caos/zitadel/internal/query"
	"github.com/caos/zitadel/pkg/grpc/auth"
)

func BulkSetMetadataToDomain(req *auth.BulkSetMyMetadataRequest) []*domain.Metadata {
	metadata := make([]*domain.Metadata, len(req.Metadata))
	for i, data := range req.Metadata {
		metadata[i] = &domain.Metadata{
			Key:   data.Key,
			Value: data.Value,
		}
	}
	return metadata
}

func ListUserMetadataToQuery(req *auth.ListMyMetadataRequest) (*query.UserMetadataSearchQueries, error) {
	offset, limit, asc := object.ListQueryToModel(req.Query)
	queries, err := metadata.MetadataQueriesToQuery(req.Queries)
	if err != nil {
		return nil, err
	}
	return &query.UserMetadataSearchQueries{
		SearchRequest: query.SearchRequest{
			Offset: offset,
			Limit:  limit,
			Asc:    asc,
		},
		Queries: queries,
	}, nil
}
