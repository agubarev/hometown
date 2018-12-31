package usermanager

import "github.com/oklog/ulid"

// Store is the storage proxy container
type Store struct {
	ds  DomainStore
	us  UserStore
	gs  GroupStore
	aps AccessPolicyStore
}

// NewStore is a shorthand for convenience
func NewStore(ds DomainStore, us UserStore, gs GroupStore, aps AccessPolicyStore) Store {
	return Store{
		ds:  ds,
		us:  us,
		gs:  gs,
		aps: aps,
	}
}

// GroupRelationKey is a shorthand for convenience
func GroupRelationKey(groupID ulid.ULID, userID ulid.ULID) []byte {
	key := make([]byte, 32)
	copy(key[0:16], groupID[0:16])
	copy(key[16:32], userID[0:16])
	return key
}

// BreakRelationKey breaks the concatenated key into individual group and user ids
func BreakRelationKey(k []byte) (ulid.ULID, ulid.ULID, error) {
	var groupID, userID ulid.ULID

	if err := groupID.Scan(k[0:16]); err != nil {
		return groupID, userID, err
	}

	if err := userID.Scan(k[16:]); err != nil {
		return groupID, userID, err
	}

	return groupID, userID, nil
}
