package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"convertpdfgo/pkg/jwt"
)

const (
	ctxUserIDKey = "user_id"
	ctxRoleKey   = "role"
)

// ===== Helper: tokenni bir nechta joydan olish (Authorization, query, cookie) =====
func extractBearerToken(c *gin.Context) string {
	// 1) Authorization: Bearer <token>
	if ah := c.GetHeader("Authorization"); ah != "" {
		parts := strings.Fields(ah)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != "" {
			return parts[1]
		}
	}
	// 2) Swagger qulayligi uchun: ?token=<jwt>
	if t := c.Query("token"); t != "" {
		return t
	}
	// 3) Cookie (agar ishlatsang)
	if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
		return cookie
	}
	return ""
}

// ===== Helper: claimsdan user_id va role ni olish, contextga qo‘yish =====
func setAuthContextFromToken(c *gin.Context, token string) error {
	claims, err := jwt.ExtractClaims(token)
	if err != nil {
		return err
	}

	uid, _ := claims["user_id"].(string)
	if uid == "" {
		return &authErr{msg: "user_id not found in token"}
	}
	// role nomi: `role` yoki legacy `user_role`
	role, _ := claims["role"].(string)
	if role == "" {
		role, _ = claims["user_role"].(string)
	}

	c.Set(ctxUserIDKey, uid)
	if role != "" {
		c.Set(ctxRoleKey, role)
	}
	return nil
}

type authErr struct{ msg string }

func (e *authErr) Error() string { return e.msg }

// ====== 1) AuthOptional: token bo‘lsa o‘qiydi, bo‘lmasa guest ======
func (h Handler) AuthOptional(c *gin.Context) {
	if tok := extractBearerToken(c); tok != "" {
		_ = setAuthContextFromToken(c, tok) // xato bo‘lsa ham guest sifatida davom
	}
	c.Next()
}

// ====== 2) AuthRequired: token majburiy ======
func (h Handler) AuthRequired(c *gin.Context) {
	tok := extractBearerToken(c)
	if tok == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}
	if err := setAuthContextFromToken(c, tok); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}
	c.Next()
}

// ====== 3) RoleGuard: ruxsat etilgan rollar ro‘yxati ======
func (h Handler) RoleGuard(allowed ...string) gin.HandlerFunc {
	allowedSet := map[string]struct{}{}
	for _, r := range allowed {
		allowedSet[strings.ToLower(strings.TrimSpace(r))] = struct{}{}
	}
	return func(c *gin.Context) {
		roleVal, ok := c.Get(ctxRoleKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role missing"})
			return
		}
		role, _ := roleVal.(string)
		if _, ok := allowedSet[strings.ToLower(role)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

// ===== Qo‘shimcha helperlar (kerak bo‘lsa) =====
func GetUserID(c *gin.Context) string {
	return c.GetString(ctxUserIDKey)
}
func GetRole(c *gin.Context) string {
	return c.GetString(ctxRoleKey)
}
