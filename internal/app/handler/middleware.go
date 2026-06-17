package handler

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"metoda/internal/app/auth"
)

const (
	AuthCookieName = "auth_token"
	ctxUserID      = "auth_user_id"
	ctxIsModerator = "auth_is_moderator"
)

func bearerPrefix() string { return "Bearer " }

func corsAllowedOriginsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isAllowedCORSOrigin(origin string, extra []string) bool {
	if origin == "" {
		return true
	}
	static := []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"https://localhost:3000",
		"https://127.0.0.1:3000",
		"http://localhost:4173",
		"http://127.0.0.1:4173",
		"http://192.168.1.35:3000",
		"http://192.168.1.35:4173",
		"https://192.168.194.69:3000",
		"https://192.168.194.69:4173",
		"http://192.168.194.69:8080",
		"http://192.168.194.69:9090",
	}
	for _, o := range static {
		if origin == o {
			return true
		}
	}
	for _, o := range extra {
		if origin == o {
			return true
		}
	}
	if strings.HasPrefix(origin, "https://") && strings.HasSuffix(origin, ".github.io") {
		return true
	}
	return false
}

func CORSMiddleware() gin.HandlerFunc {
	extra := corsAllowedOriginsFromEnv()
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Authorization")
		c.Writer.Header().Set("Vary", "Origin")

		if origin != "" && isAllowedCORSOrigin(origin, extra) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func extractJWT(r *http.Request) string {
	if c, err := r.Cookie(AuthCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	h := r.Header.Get("Authorization")
	p := bearerPrefix()
	if h != "" && strings.HasPrefix(h, p) {
		return strings.TrimPrefix(h, p)
	}
	return ""
}

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractJWT(c.Request)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			return
		}
		blacklisted, err := h.Repository.IsTokenBlacklisted(context.Background(), tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"status": "error", "description": err.Error()})
			return
		}
		if blacklisted {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			return
		}
		claims, err := auth.ParseAndValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			return
		}
		uid, err := auth.UserIDFromClaims(claims)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			return
		}
		h.Repository.SetUserID(int(uid))
		c.Set(ctxUserID, uid)
		c.Set(ctxIsModerator, auth.IsModeratorFromClaims(claims))
		c.Next()
	}
}

func (h *Handler) RequireModerator() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(ctxIsModerator)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "forbidden"})
			return
		}
		isMod, ok := v.(bool)
		if !ok || !isMod {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "forbidden"})
			return
		}
		c.Next()
	}
}

func authUserIDUint(ctx *gin.Context) (uint, error) {
	v, ok := ctx.Get(ctxUserID)
	if !ok {
		return 0, errors.New("нет user_id в контексте")
	}
	id, ok := v.(uint)
	if !ok || id == 0 {
		return 0, errors.New("неверный user_id")
	}
	return id, nil
}

func isModeratorFromCtx(ctx *gin.Context) bool {
	v, ok := ctx.Get(ctxIsModerator)
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}
