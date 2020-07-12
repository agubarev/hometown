package user

import (
	"context"
	"net/http"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/internal/server/endpoints"
)

func Get(ctx context.Context, c *core.Core, w http.ResponseWriter, r *http.Request) (result interface{}, code int, err error) {
	u, err := c.UserManager().UserByID(
		ctx,
		r.Context().Value(endpoints.CKUserID).(int64),
	)

	if err != nil {
		return nil, http.StatusForbidden, err
	}

	return u, http.StatusOK, nil
}
