package usermanager_test

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/usermanager"
)

func init() {
	confstring := `
instance:
    domains:
        directory: /tmp/hometown/domains
`

	viper.SetConfigType("yaml")

	if err := viper.ReadConfig(strings.NewReader(confstring)); err != nil {
		panic(err)
	}
}

func TestNewService(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	m, err := usermanager.New()
	a.NoError(err)
	a.NotNil(m)

	s, err := usermanager.NewUserService(m)
	a.NoError(err)
	a.NotNil(s)
}

func TestServiceRegister(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	m, err := usermanager.New()
	a.NoError(err)
	a.NotNil(m)

	s, err := usermanager.NewUserService(m)
	a.NoError(err)
	a.NotNil(s)

	// TODO: implement
}
