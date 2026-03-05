package middleware

import (
	"aATA/pkg/jwt"
	"strings"

	"github.com/gin-gonic/gin"
)

type JWTMid struct {
	jwt *jwt.JWT
}

func NewJWTMid(j *jwt.JWT) *JWTMid {
	return &JWTMid{
		jwt: j,
	}
}

func (m *JWTMid) Handler(ctx *gin.Context) {
	// 处理校验，检查 Token，若不通过则立即停止
	auth := ctx.GetHeader("Authorization") // HTTP Header 约定俗成的标准是将 Token 放在 Header 的 Authorization
	if auth == "" {
		ctx.JSON(401, gin.H{"msg": "未登录"})
		ctx.Abort() // 立即停止
		return
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		ctx.JSON(401, gin.H{"msg": "Token 格式错误"})
		ctx.Abort()
		return
	}

	// 使用 jwt 中间件从 Token 中提取信息，以 map 的形式传回来
	claims, err := m.jwt.ParseToken(parts[1])
	if err != nil {
		ctx.JSON(401, gin.H{"msg": "Token 无效或已过期"})
		ctx.Abort()
		return
	}

	// 将解析出来的数据写入 ctx 中
	// 1. 处理 uid
	uidFloat, ok := claims["uid"].(string)
	if !ok {
		ctx.JSON(401, gin.H{"msg": "Token 数据异常"})
		ctx.Abort()
		return
	}
	ctx.Set("uid", string(uidFloat))

	// 2. 处理 isAdmin
	isAdmin := false
	if v, ok := claims["is_admin"]; ok {
		if b, ok := v.(bool); ok {
			isAdmin = b
		}
	}
	ctx.Set("is_admin", isAdmin)

	// 3. 写入 name
	if name, ok := claims["name"].(string); ok {
		ctx.Set("name", name)
	}

	ctx.Next() // 批准放行
}
