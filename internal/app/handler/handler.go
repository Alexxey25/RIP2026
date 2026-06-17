package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"metoda/internal/app/repository"
)

// URL превью файлов; порт должен совпадать с MINIO_PORT (.env) и docker-compose (nginx).
const minioBaseURL = "http://192.168.194.69:9090/constructions"

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
	}
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.Use(CORSMiddleware())

	router.GET("/", h.GetConstructions)
	router.GET("/construction/:id", h.GetConstruction)
	router.GET("/dendrochronology", h.GetDendrochronology)
	router.POST("/dendrochronology/update-item", h.UpdateDendrochronologyItem)
	router.POST("/form-dendrochronology", h.FormDendrochronology)
	router.POST("/add-to-dendrochronology", h.AddToDendrochronology)
	router.POST("/delete-dendrochronology", h.DeleteDendrochronology)

	api := router.Group("/api")

	// Без авторизации: чтение каталога + регистрация и вход
	api.GET("/constructions", h.APIGetConstructions)
	api.GET("/constructions/:id", h.APIGetConstruction)
	api.POST("/users/signup", h.APISignUp)
	api.POST("/users/signin", h.APISignIn)

	needAuth := api.Group("")
	needAuth.Use(h.AuthMiddleware())
	needAuth.POST("/users/signout", h.APISignOut)
	needAuth.GET("/dendrochronologies/cart", h.APIGetCart)
	needAuth.GET("/dendrochronologies", h.APIGetDendrochronologies)
	needAuth.GET("/dendrochronologies/:id", h.APIGetDendrochronology)
	needAuth.PUT("/dendrochronologies/:id/form", h.APIFormDendrochronology)
	needAuth.PUT("/dendrochronologies/:id", h.APIUpdateDendrochronology)
	needAuth.DELETE("/dendrochronologies/:id", h.APIDeleteDendrochronology)
	needAuth.POST("/constructions/:id/add-to-dendrochronology", h.APIAddToCart)
	needAuth.DELETE("/dendrochronology-constructions/:construction_id/:dendrochronology_id", h.APIDeleteFromCart)
	needAuth.PUT("/dendrochronology-constructions/:construction_id/:dendrochronology_id", h.APIUpdateCartItem)

	mod := api.Group("")
	mod.Use(h.AuthMiddleware())
	mod.Use(h.RequireModerator())
	mod.POST("/constructions", h.APICreateConstruction)
	mod.PUT("/dendrochronologies/:id/finish", h.APIFinishDendrochronology)
}

func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./resources")
}

func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, err error) {
	logrus.Error(err.Error())
	ctx.JSON(errorStatusCode, gin.H{
		"status":      "error",
		"description": err.Error(),
	})
}
