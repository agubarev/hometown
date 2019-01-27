package usermanager_test

import (
	"strings"
	"testing"

	"gitlab.com/agubarev/hometown/usermanager"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func init() {
	cstr := `
instance:
    domains:
        directory: ../data/test/domains
`

	viper.SetConfigType("yaml")

	if err := viper.ReadConfig(strings.NewReader(cstr)); err != nil {
		panic(err)
	}
}

func TestNewDomain(t *testing.T) {
	a := assert.New(t)

	owner, err := usermanager.NewUser("dummy", "dummy@example.com")
	a.NoError(err)
	a.NotNil(owner)

	d, err := usermanager.NewDomain(owner, nil)
	a.NoError(err)
	a.NotNil(d)
}
