package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"metoda/internal/app/ds"
)

const creatorID = uint(1)

func (h *Handler) GetConstructions(ctx *gin.Context) {
	var constructions []ds.Construction
	var err error

	searchQuery := ctx.Query("query")
	if searchQuery == "" {
		constructions, err = h.Repository.GetAllConstructions()
		if err != nil {
			logrus.Error(err)
		}
	} else {
		constructions, err = h.Repository.SearchConstructionsByTitle(searchQuery)
		if err != nil {
			logrus.Error(err)
		}
	}

	cartCount := h.Repository.GetCartCount(creatorID)
	draftID := h.Repository.GetDraftDendrochronologyID(creatorID)

	ctx.HTML(http.StatusOK, "mainpage.html", gin.H{
		"constructions": constructions,
		"query":         searchQuery,
		"cartCount":     cartCount,
		"draftID":       draftID,
		"minioBase":     minioBaseURL,
	})
}

func (h *Handler) GetConstruction(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/")
		return
	}

	construction, err := h.Repository.GetConstructionByID(id)
	if err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/")
		return
	}

	if construction == nil {
		ctx.Redirect(http.StatusFound, "/")
		return
	}

	ctx.HTML(http.StatusOK, "productpage.html", gin.H{
		"construction": construction,
		"minioBase":    minioBaseURL,
	})
}

