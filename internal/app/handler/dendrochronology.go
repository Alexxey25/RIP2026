package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *Handler) GetDendrochronology(ctx *gin.Context) {
	draft, err := h.Repository.GetDraftDendrochronology(creatorID)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/")
		return
	}

	_, views, err := h.Repository.GetDendrochronologyWithConstructions(draft.ID)
	if err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/")
		return
	}

	totalSamples := h.Repository.GetTotalSamples(draft.ID)
	// На странице черновика всегда показываем дату по данным корзины (м-м), чтобы при изменении полей и Enter значение пересчитывалось
	buildYear := h.Repository.GetEstimatedBuildYear(draft.ID)

	ctx.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	ctx.Header("Pragma", "no-cache")
	ctx.HTML(http.StatusOK, "dendrochronologypage.html", gin.H{
		"dendrochronology": draft,
		"constructions":    views,
		"totalSamples":     totalSamples,
		"buildYear":        buildYear,
		"minioBase":        minioBaseURL,
	})
}

func (h *Handler) AddToDendrochronology(ctx *gin.Context) {
	strId := ctx.PostForm("construction_id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.Repository.AddConstructionToDendrochronology(uint(id), creatorID)
	if err != nil && !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		logrus.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Redirect(http.StatusFound, "/")
}

// UpdateDendrochronologyItem обрабатывает обновление одной строки корзины:
// количество образцов, дата рубки и поправка на дату.
func (h *Handler) UpdateDendrochronologyItem(ctx *gin.Context) {
	strId := ctx.PostForm("item_id")
	itemID, err := strconv.Atoi(strId)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	samplesStr := ctx.PostForm("samples_count")
	samples := 1
	if samplesStr != "" {
		if v, convErr := strconv.Atoi(samplesStr); convErr == nil && v > 0 {
			samples = v
		}
	}

	cuttingDate := strings.TrimSpace(ctx.PostForm("cutting_date"))
	dateCorrection := strings.TrimSpace(ctx.PostForm("date_correction"))

	err = h.Repository.UpdateDendrochronologyItem(uint(itemID), samples, cuttingDate, dateCorrection)
	if err != nil {
		logrus.Error(err)
	}
	// Редирект с 303 заставляет браузер сделать новый GET, без подстановки кэша
	ctx.Redirect(http.StatusSeeOther, "/dendrochronology")
}

func (h *Handler) FormDendrochronology(ctx *gin.Context) {
	strId := ctx.PostForm("dendrochronology_id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/dendrochronology")
		return
	}

	err = h.Repository.FormDendrochronology(uint(id))
	if err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/dendrochronology")
		return
	}

	ctx.Redirect(http.StatusFound, "/")
}

func (h *Handler) DeleteDendrochronology(ctx *gin.Context) {
	strId := ctx.PostForm("dendrochronology_id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/")
		return
	}

	err = h.Repository.DeleteDendrochronologyBySQL(uint(id))
	if err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/dendrochronology")
		return
	}

	ctx.Redirect(http.StatusFound, "/")
}
