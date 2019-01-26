package usermanager

import "github.com/oklog/ulid"

// GroupRelationKey is a shorthand for convenience
// TODO: rename to just RelationKey
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
