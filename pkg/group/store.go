package group

import (
	"context"

	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/google/uuid"
)

// Store describes a storage contract for groups specifically
type Store interface {
	UpsertGroup(ctx context.Context, g Group) (Group, error)
	CreateRelation(ctx context.Context, rel Relation) error
	FetchGroupByID(ctx context.Context, groupID uuid.UUID) (g Group, err error)
	FetchGroupByKey(ctx context.Context, key bytearray.ByteString32) (g Group, err error)
	FetchGroupByName(ctx context.Context, name bytearray.ByteString128) (g Group, err error)
	FetchGroupsByName(ctx context.Context, isPartial bool, name bytearray.ByteString128) (gs []Group, err error)
	HasRelation(ctx context.Context, rel Relation) (bool, error)
	FetchAllGroups(ctx context.Context) (gs []Group, err error)
	FetchAllRelations(ctx context.Context) ([]Relation, error)
	FetchGroupRelations(ctx context.Context, groupID uuid.UUID) ([]Relation, error)
	DeleteByID(ctx context.Context, groupID uuid.UUID) error
	DeleteRelation(ctx context.Context, rel Relation) error
}
