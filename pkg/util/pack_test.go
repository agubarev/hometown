package util_test

import (
	"testing"

	"github.com/agubarev/hometown/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestPackUnpackU32s(t *testing.T) {
	a := assert.New(t)

	n1 := uint32(67816)
	n2 := uint32(1982)
	n3 := (uint64(n1) << 32) | uint64(n2)

	a.Equal(util.PackU32s(^uint32(0), ^uint32(0)), ^uint64(0))
	a.Equal(util.PackU32s(n1, n2), n3)

	a.Equal(util.UnpackU32s(util.PackU32s(n1, n2)), [2]uint32{n1, n2})
	a.Equal(util.UnpackU32s(n3), [2]uint32{n1, n2})
}
