package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	CtxUserID = "userId"
	CtxOpenid = "openid"
)

// AuthUser 当前登录用户
type AuthUser struct {
	UserID string
	Openid string
}

func parseToken(c *gin.Context, secret string) (*AuthUser, bool) {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, false
	}
	tokenStr := strings.TrimPrefix(h, "Bearer ")
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, false
	}
	sub, _ := claims["sub"].(string)
	openid, _ := claims["openid"].(string)
	if sub == "" {
		return nil, false
	}
	return &AuthUser{UserID: sub, Openid: openid}, true
}

// JWTAuth 必须登录
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := parseToken(c, secret)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "code": "UNAUTHORIZED", "message": "请重新登录",
			})
			return
		}
		c.Set(CtxUserID, u.UserID)
		c.Set(CtxOpenid, u.Openid)
		c.Next()
	}
}

// OptionalJWT 有 token 解析、无 token 放行（访客匿名浏览用）
func OptionalJWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if u, ok := parseToken(c, secret); ok {
			c.Set(CtxUserID, u.UserID)
			c.Set(CtxOpenid, u.Openid)
		}
		c.Next()
	}
}

// Current 从上下文取当前用户（可能为空）
func Current(c *gin.Context) *AuthUser {
	uid, _ := c.Get(CtxUserID)
	openid, _ := c.Get(CtxOpenid)
	uidStr, _ := uid.(string)
	openidStr, _ := openid.(string)
	if uidStr == "" && openidStr == "" {
		return nil
	}
	return &AuthUser{UserID: uidStr, Openid: openidStr}
}
