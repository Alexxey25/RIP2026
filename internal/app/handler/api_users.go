package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"metoda/internal/app/auth"
	"metoda/internal/app/repository"
	"metoda/internal/app/serializer"
)

// APISignUp Регистрация
// @Summary Регистрация пользователя
// @Description Создание учётной записи (без авторизации).
// @Tags users
// @Accept json
// @Produce json
// @Param body body serializer.UserJSON true "Логин и пароль"
// @Success 201 {object} serializer.UserJSON
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/signup [post]
func (h *Handler) APISignUp(ctx *gin.Context) {
	var j serializer.UserJSON
	if err := ctx.ShouldBindJSON(&j); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	if j.Login == "" {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("поле login обязательно"))
		return
	}
	if j.Password == "" {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("поле password обязательно"))
		return
	}

	u, err := h.Repository.CreateUser(j)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			h.errorHandler(ctx, http.StatusConflict, err)
		} else {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.Header("Location", fmt.Sprintf("/api/users/%d", u.ID))
	ctx.JSON(http.StatusCreated, serializer.UserToJSON(u))
}

// APISignIn Вход: JWT в теле и в HttpOnly cookie (сессия)
// @Summary Вход в систему
// @Description Возвращает token и пользователя; устанавливает cookie auth_token.
// @Tags users
// @Accept json
// @Produce json
// @Param body body serializer.UserJSON true "Логин и пароль"
// @Success 200 {object} map[string]interface{} "token, user"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/signin [post]
func (h *Handler) APISignIn(ctx *gin.Context) {
	var j serializer.UserJSON
	if err := ctx.ShouldBindJSON(&j); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	if j.Login == "" {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("поле login обязательно"))
		return
	}
	if j.Password == "" {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("поле password обязательно"))
		return
	}

	u, token, err := h.Repository.SignInWithToken(j)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}

	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     AuthCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	ctx.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  serializer.UserToJSON(u),
	})
}

// APISignOut Выход: отзыв JWT в Redis (blacklist) и сброс cookie
// @Summary Выход
// @Description Требует Authorization: Bearer или cookie auth_token.
// @Tags users
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security ApiKeyAuth
// @Router /users/signout [post]
func (h *Handler) APISignOut(ctx *gin.Context) {
	tokenString := extractJWT(ctx.Request)
	if tokenString == "" {
		h.errorHandler(ctx, http.StatusUnauthorized, errors.New("нет токена"))
		return
	}

	claims, err := auth.ParseAndValidateToken(tokenString)
	if err != nil {
		h.clearAuthCookie(ctx)
		ctx.JSON(http.StatusOK, gin.H{"status": "signed_out"})
		return
	}

	ttl, err := tokenTTLFromClaims(claims)
	if err != nil {
		ttl = time.Hour
	}
	if err := h.Repository.AddTokenToBlacklist(context.Background(), tokenString, ttl); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	h.clearAuthCookie(ctx)
	h.Repository.SignOut()
	ctx.JSON(http.StatusOK, gin.H{"status": "signed_out"})
}

func (h *Handler) clearAuthCookie(ctx *gin.Context) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func tokenTTLFromClaims(claims jwt.MapClaims) (time.Duration, error) {
	expVal, ok := claims["exp"]
	if !ok {
		return 0, errors.New("exp not present")
	}
	var expUnix int64
	switch v := expVal.(type) {
	case float64:
		expUnix = int64(v)
	case int64:
		expUnix = v
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, err
		}
		expUnix = i
	default:
		return 0, fmt.Errorf("unsupported exp type %T", v)
	}
	expTime := time.Unix(expUnix, 0)
	ttl := time.Until(expTime)
	if ttl < 0 {
		return 0, errors.New("token already expired")
	}
	return ttl, nil
}
