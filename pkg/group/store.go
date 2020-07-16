package group

import (
	"context"

	"github.com/google/uuid"
)

// Store describes a storage contract for groups specifically
type Store interface {
	UpsertGroup(ctx context.Context, g Group) (Group, error)
	CreateRelation(ctx context.Context, groupID uuid.UUID, assetKind AssetKind, assetID uuid.UUID) error
	FetchGroupByID(ctx context.Context, groupID uuid.UUID) (g Group, err error)
	FetchGroupByKey(ctx context.Context, key TKey) (g Group, err error)
	FetchGroupByName(ctx context.Context, name TName) (g Group, err error)
	FetchGroupsByName(ctx context.Context, isPartial bool, name TName) (gs []Group, err error)
	FetchAllGroups(ctx context.Context) (gs []Group, err error)
	FetchAllRelations(ctx context.Context) (map[uuid.UUID][]uuid.UUID, error)
	FetchGroupRelations(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	HasRelation(ctx context.Context, groupID uuid.UUID, assetKind AssetKind, assetID uuid.UUID) (bool, error)
	DeleteByID(ctx context.Context, groupID uuid.UUID) error
	DeleteRelation(ctx context.Context, groupID uuid.UUID, assetKind AssetKind, assetID uuid.UUID) error
}
