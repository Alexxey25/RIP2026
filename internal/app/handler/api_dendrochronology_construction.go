package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"metoda/internal/app/repository"
	"metoda/internal/app/serializer"
)

// APIAddToCart Добавить конструкцию в корзину (черновик)
// @Summary Добавить в корзину
// @Description Добавляет конструкцию в черновик заявки; при первом добавлении создаётся заявка (201 + Location).
// @Tags dendrochronologies
// @Produce json
// @Param id path int true "ID конструкции"
// @Security ApiKeyAuth
// @Success 200 {object} serializer.DendrochronologyListJSON
// @Success 201 {object} serializer.DendrochronologyListJSON
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /constructions/{id}/add-to-dendrochronology [post]
func (h *Handler) APIAddToCart(ctx *gin.Context) {
	constructionID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный id конструкции"))
		return
	}

	uid, err := authUserIDUint(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}
	d, created, err := h.Repository.AddConstructionToCartAPI(uint(constructionID), uid)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.errorHandler(ctx, http.StatusNotFound, err)
		} else if errors.Is(err, repository.ErrAlreadyExists) {
			h.errorHandler(ctx, http.StatusConflict, err)
		} else {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	status := http.StatusOK
	if created {
		ctx.Header("Location", fmt.Sprintf("/api/dendrochronologies/%d", d.ID))
		status = http.StatusCreated
	}

	creatorLogin := h.Repository.GetCreatorLogin(d.CreatorID)
	moderatorLogin := h.Repository.GetModeratorLogin(d.ModeratorID)
	datedCount := h.Repository.GetDatedConstructionsCount(d.ID)
	ctx.JSON(status, serializer.DendrochronologyToListJSON(d, creatorLogin, moderatorLogin, datedCount))
}

// APIDeleteFromCart Убрать конструкцию из заявки
// @Summary Удалить строку из заявки
// @Tags dendrochronologies
// @Produce json
// @Param construction_id path int true "ID конструкции"
// @Param dendrochronology_id path int true "ID заявки"
// @Security ApiKeyAuth
// @Success 200 {object} serializer.DendrochronologyListJSON
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dendrochronology-constructions/{construction_id}/{dendrochronology_id} [delete]
func (h *Handler) APIDeleteFromCart(ctx *gin.Context) {
	constructionID, err := strconv.Atoi(ctx.Param("construction_id"))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный construction_id"))
		return
	}
	dendrochronologyID, err := strconv.Atoi(ctx.Param("dendrochronology_id"))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный dendrochronology_id"))
		return
	}

	uid, err := authUserIDUint(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}
	d, err := h.Repository.DeleteConstructionFromCartAPI(constructionID, dendrochronologyID, int(uid))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.errorHandler(ctx, http.StatusNotFound, err)
		} else if errors.Is(err, repository.ErrNotAllowed) {
			h.errorHandler(ctx, http.StatusForbidden, err)
		} else {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	creatorLogin := h.Repository.GetCreatorLogin(d.CreatorID)
	moderatorLogin := h.Repository.GetModeratorLogin(d.ModeratorID)
	datedCount := h.Repository.GetDatedConstructionsCount(d.ID)
	ctx.JSON(http.StatusOK, serializer.DendrochronologyToListJSON(d, creatorLogin, moderatorLogin, datedCount))
}

// APIUpdateCartItem Обновить параметры строки в заявке
// @Summary Изменить строку заявки
// @Description Образцы, даты резки и коррекции для пары конструкция–заявка
// @Tags dendrochronologies
// @Accept json
// @Produce json
// @Param construction_id path int true "ID конструкции"
// @Param dendrochronology_id path int true "ID заявки"
// @Param body body serializer.DendroConstructionUpdateJSON true "Поля"
// @Security ApiKeyAuth
// @Success 200 {object} serializer.DendroConstructionUpdateJSON
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dendrochronology-constructions/{construction_id}/{dendrochronology_id} [put]
func (h *Handler) APIUpdateCartItem(ctx *gin.Context) {
	constructionID, err := strconv.Atoi(ctx.Param("construction_id"))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный construction_id"))
		return
	}
	dendrochronologyID, err := strconv.Atoi(ctx.Param("dendrochronology_id"))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный dendrochronology_id"))
		return
	}

	var j serializer.DendroConstructionUpdateJSON
	if err := ctx.ShouldBindJSON(&j); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	uid, err := authUserIDUint(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}
	item, err := h.Repository.UpdateConstructionInCartAPI(constructionID, dendrochronologyID, j, int(uid))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.errorHandler(ctx, http.StatusNotFound, err)
		} else if errors.Is(err, repository.ErrNotAllowed) {
			h.errorHandler(ctx, http.StatusForbidden, err)
		} else {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	life := strings.TrimSpace(item.UseLifeOverride)
	if life == "" {
		life = item.Construction.UseLife
	}

	ctx.JSON(http.StatusOK, serializer.DendroConstructionUpdateJSON{
		SamplesCount:   item.SamplesCount,
		CuttingDate:    item.CuttingDate,
		DateCorrection: item.DateCorrection,
		UseLife:        life,
	})
}
