package group

import (
	"context"
	"net/http"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/security/auth"
	"github.com/agubarev/hometown/pkg/user"
)

func Post(ctx context.Context, c *core.Core, ac auth.Context, w http.ResponseWriter, r *http.Request) (result interface{}, code int, err error) {
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	newUser := user.NewUserObject{
		Essential:        user.Essential{},
		ProfileEssential: user.ProfileEssential{},
		EmailAddr:        "",
		PhoneNumber:      "",
		Password:         nil,
	}

	return newUser, http.StatusOK, nil
}
