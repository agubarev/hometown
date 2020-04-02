package endpoints

import (
	"context"
	"net/http"

	"github.com/agubarev/hometown/internal/core"
)

func UserGet(ctx context.Context, c *core.Core, w http.ResponseWriter, r *http.Request) (result interface{}, code int, err error) {
	u, err := c.UserManager().UserByID(
		ctx,
		r.Context().Value(keyUserID).(int64),
	)

	if err != nil {
		return nil, http.StatusForbidden, err
	}

	return u, http.StatusOK, nil
}
