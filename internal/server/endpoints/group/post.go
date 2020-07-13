package group

import (
	"context"
	"net/http"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/internal/server"
	"github.com/agubarev/hometown/internal/server/endpoints"
	"github.com/agubarev/hometown/pkg/user"
)

func Post(ctx context.Context, c *core.Core, w http.ResponseWriter, r *http.Request) (result interface{}, code int, err error) {
	c := ctx.Value(server.CKCore)

	newUser := user.NewUserObject{
		Essential:        user.Essential{},
		ProfileEssential: user.ProfileEssential{},
		EmailAddr:        "",
		PhoneNumber:      "",
		Password:         nil,
	}

	/*
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	*/

	return newUser, http.StatusOK, nil
}
