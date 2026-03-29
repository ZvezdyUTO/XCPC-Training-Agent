package app

import (
	"aATA/internal/app/apperr"
	"fmt"

	"aATA/pkg/httpx"

	"github.com/gin-gonic/gin"
)

func InitErrorHandler() {
	httpx.SetErrorHandler(func(ctx *gin.Context, err error) (int, error) {
		appErr, ok := apperr.As(err)
		if ok {
			if appErr.Kind == apperr.KindInternal {
				fmt.Println("[internal error]", err)
				return 500, apperr.New(apperr.KindInternal, "internal_error", "internal server error", 500)
			}
			if appErr.HTTPStatus > 0 {
				return appErr.HTTPStatus, appErr
			}
			return 500, appErr
		}

		fmt.Println("[internal error]", err)
		return 500, apperr.New(apperr.KindInternal, "internal_error", "internal server error", 500)
	})
}
