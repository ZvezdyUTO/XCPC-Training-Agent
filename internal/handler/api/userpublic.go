package api

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"aATA/internal/domain"
	"aATA/internal/logic"
	"aATA/internal/svc"
	"aATA/pkg/httpx"
)

type UserPublic struct {
	svcCtx *svc.ServiceContext
	user   logic.User
}

func NewUserPublic(svcCtx *svc.ServiceContext, user logic.User) *UserPublic {
	return &UserPublic{
		svcCtx: svcCtx,
		user:   user,
	}
}

func (h *UserPublic) InitRegister(engine *gin.Engine) {
	g := engine.Group("v1/user")
	g.POST("/login", h.Login)
	g.POST("/register", h.Register)
}

func (h *UserPublic) Login(ctx *gin.Context) {
	var req domain.LoginReq
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	res, err := h.user.Login(ctx.Request.Context(), &req)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	token, err := h.svcCtx.JWT.GenerateToken(map[string]interface{}{ // 登录时在handler处生成token
		"uid":      res.Id,
		"name":     res.Name,
		"status":   res.Status,
		"is_admin": res.IsSystem == 1,
	})
	fmt.Println(res.IsSystem)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.OkWithData(ctx, gin.H{
		"user":  res,
		"token": token,
	})
}

func (h *UserPublic) Register(ctx *gin.Context) {
	var req domain.RegisterReq
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	res, err := h.user.Register(ctx.Request.Context(), &req)
	if err != nil {
		httpx.FailWithErr(ctx, err)
	} else {
		httpx.OkWithData(ctx, res)
	}
}
