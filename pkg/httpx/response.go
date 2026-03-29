package httpx

import (
	"aATA/internal/app/apperr"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
)

const (
	ERROR   = 500
	SUCCESS = 200
)

var (
	ERRORMSG   = errors.New("fail")
	SUCCESSMSG = "success"
)

var (
	errorHandler func(ctx *gin.Context, err error) (int, error) // 传输上下文和错误，返回错误
	okHandler    func(ctx *gin.Context, data any) any           // 传输上下文和信息，返回任何信息
)

var NULL = map[string]interface{}{}

// 自定义返回值

func SetOkHandler(handler func(ctx *gin.Context, data any) any) {
	okHandler = handler
}

func SetErrorHandler(handler func(ctx *gin.Context, err error) (int, error)) {
	errorHandler = handler
}

type Response struct { // 统一响应结构
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Msg     string      `json:"msg"`
	ErrCode string      `json:"err_code,omitempty"`
}

func Result(ctx *gin.Context, httpCode int, bizCode int, data interface{}, msg string, errCode string) {
	ctx.JSON(httpCode, &Response{
		Code:    bizCode,
		Data:    data,
		Msg:     msg,
		ErrCode: errCode,
	})
}

// 两种执行情况：OK Fail，以及是否返回信息

func Ok(ctx *gin.Context) {
	OkWithData(ctx, NULL)
}

func OkWithData(ctx *gin.Context, data interface{}) {
	handler := okHandler
	if handler != nil {
		data = handler(ctx, data)
	}
	Result(ctx, 200, SUCCESS, data, SUCCESSMSG, "")
}

func Fail(ctx *gin.Context) {
	FailWithErr(ctx, ERRORMSG)
}

func FailWithErr(ctx *gin.Context, err error) {
	handler := errorHandler

	httpCode := 500
	bizCode := 50000
	msg := "internal server error"
	errCode := ""

	if handler != nil {
		httpCode, err = handler(ctx, err)
	}

	if appErr, ok := apperr.As(err); ok {
		msg = appErr.Message
		errCode = appErr.Code
	} else if err != nil {
		fmt.Println("[internal error]", err)
		msg = err.Error()
	}

	Result(ctx, httpCode, bizCode, NULL, msg, errCode)
}
