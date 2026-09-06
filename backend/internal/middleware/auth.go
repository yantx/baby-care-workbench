package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/yantx/baby-care-workbench/backend/internal/config"
	"github.com/yantx/baby-care-workbench/backend/internal/ws"
)

const (
	CtxUserID   = "user_id"
	CtxFamilyID = "family_id"
)

// JWTAuth Bearer Token 认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			abortUnauthorized(c, "缺少认证信息")
			return
		}
		claims, err := ws.ParseToken(strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			abortUnauthorized(c, "token 无效或已过期")
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxFamilyID, claims.FamilyID)
		c.Next()
}
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code":    40101,
		"message": msg,
	})
}

// IssueToken 签发 JWT（含活跃家庭，供 REST 鉴权与 WS 连接复用）
func IssueToken(userID, familyID int64) (token string, expires int64, err error) {
	claims := ws.WSClaims{
		UserID:   userID,
		FamilyID: familyID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.Get().JWT.ExpireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = t.SignedString([]byte(config.Get().JWT.Secret))
	expires = claims.ExpiresAt.Time.Unix()
	return
}
