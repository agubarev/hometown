package group

import "context"

// Store describes a storage contract for groups specifically
type Store interface {
	UpsertGroup(ctx context.Context, g Group) (Group, error)
	CreateRelation(ctx context.Context, groupID, userID uint32) error
	FetchGroupByID(ctx context.Context, groupID uint32) (g Group, err error)
	FetchGroupByKey(ctx context.Context, key TKey) (g Group, err error)
	FetchGroupByName(ctx context.Context, name TName) (g Group, err error)
	FetchGroupsByName(ctx context.Context, isPartial bool, name TName) (gs []Group, err error)
	FetchAllGroups(ctx context.Context) (gs []Group, err error)
	FetchAllRelations(ctx context.Context) (map[uint32][]uint32, error)
	FetchGroupRelations(ctx context.Context, groupID uint32) ([]uint32, error)
	HasRelation(ctx context.Context, groupID, userID uint32) (bool, error)
	DeleteByID(ctx context.Context, groupID uint32) error
	DeleteRelation(ctx context.Context, groupID, userID uint32) error
}
