package api

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"aATA/internal/domain"
	"aATA/internal/logic"
	"aATA/internal/svc"
	"aATA/pkg/httpx"
)

type AdminUser struct {
	svcCtx *svc.ServiceContext
	user   logic.User
}

func NewAdminUser(svcCtx *svc.ServiceContext, user logic.User) *AdminUser {
	return &AdminUser{
		svcCtx: svcCtx,
		user:   user,
	}
}

func (h *AdminUser) InitRegister(engine *gin.Engine) {
	// RESTful 架构，用 URL 表示资源，用 HTTP 动词表示动作
	g := engine.Group("v1/admin/users", h.svcCtx.JwtMid.Handler, h.svcCtx.AdminMid.Handler)
	g.GET("/list", h.List)
	g.POST("/create", h.Create)
	g.DELETE("/:id", h.Delete)
}

func (h *AdminUser) List(ctx *gin.Context) {
	var req domain.UserListReq
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	res, err := h.user.List(ctx.Request.Context(), &req)
	if err != nil {
		httpx.FailWithErr(ctx, err)
	} else {
		httpx.OkWithData(ctx, res)
	}
}

func (h *AdminUser) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.FailWithErr(ctx, errors.New("参数错误"))
		return
	}

	uid, _ := h.svcCtx.JWT.GetUID(ctx)
	if uid == id {
		httpx.FailWithErr(ctx, errors.New("不能删除自己"))
		return
	}

	err = h.user.AdminDelete(ctx.Request.Context(), uid, id)
	if err != nil {
		httpx.FailWithErr(ctx, err)
	} else {
		httpx.Ok(ctx)
	}
}

func (h *AdminUser) Create(ctx *gin.Context) {
	var req domain.AdminBatchCreateUsersReq
	if err := httpx.BindAndValidate(ctx, &req); err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	result, err := h.user.Create(ctx.Request.Context(), req.Users)
	if err != nil {
		httpx.FailWithErr(ctx, err)
		return
	}

	httpx.OkWithData(ctx, result)
}
