package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"metoda/internal/app/repository"
	"metoda/internal/app/serializer"
)

// APIGetCart Корзина (черновик заявки)
// @Summary Корзина (черновик)
// @Description Если черновика нет — { "status": "no_draft", "constructions_count": 0 }.
// @Tags dendrochronologies
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "CartJSON (dendrochronology_id, constructions_count) или no_draft"
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dendrochronologies/cart [get]
func (h *Handler) APIGetCart(ctx *gin.Context) {
	uid, err := authUserIDUint(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}
	id, count, err := h.Repository.GetCartInfo(uid)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}
	if id == 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"status":              "no_draft",
			"constructions_count": 0,
		})
		return
	}
	ctx.JSON(http.StatusOK, serializer.CartJSON{
		DendrochronologyID: id,
		ConstructionsCount: count,
	})
}

// APIGetDendrochronologies Список заявок
// @Summary Список заявок на дендроанализ
// @Description Все заявки; фильтры по дате формирования и статусу. Пользователь видит свои; модератор — все.
// @Tags dendrochronologies
// @Produce json
// @Param from_date query string false "Начало диапазона даты формирования, YYYY-MM-DD"
// @Param to_date query string false "Конец диапазона, YYYY-MM-DD"
// @Param status query string false "Фильтр по статусу заявки"
// @Param page query int false "Номер страницы"
// @Param limit query int false "Количество элементов на странице"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dendrochronologies [get]
func (h *Handler) APIGetDendrochronologies(ctx *gin.Context) {
	var from, to time.Time
	if s := ctx.Query("from_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный формат from_date, ожидается YYYY-MM-DD"))
			return
		}
		from = t
	}
	if s := ctx.Query("to_date"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный формат to_date, ожидается YYYY-MM-DD"))
			return
		}
		to = t
	}
	status := ctx.Query("status")
	
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	uid, err := authUserIDUint(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}
	list, total, err := h.Repository.GetAllDendrochronologies(from, to, status, uid, isModeratorFromCtx(ctx), limit, offset)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	resp := make([]serializer.DendrochronologyListJSON, 0, len(list))
	for _, d := range list {
		creatorLogin := h.Repository.GetCreatorLogin(d.CreatorID)
		moderatorLogin := h.Repository.GetModeratorLogin(d.ModeratorID)
		datedCount := h.Repository.GetDatedConstructionsCount(d.ID)
		resp = append(resp, serializer.DendrochronologyToListJSON(d, creatorLogin, moderatorLogin, datedCount))
	}
	ctx.JSON(http.StatusOK, gin.H{
		"total": total,
		"page":  page,
		"limit": limit,
		"data":  resp,
	})
}

// APIGetDendrochronology Детальная заявка
// @Summary Заявка по id
// @Description Состав конструкций и логины; доступ: создатель или модератор.
// @Tags dendrochronologies
// @Produce json
// @Param id path int true "ID заявки"
// @Security ApiKeyAuth
// @Success 200 {object} serializer.DendrochronologyDetailJSON
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dendrochronologies/{id} [get]
func (h *Handler) APIGetDendrochronology(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный id"))
		return
	}

	d, err := h.Repository.GetDendrochronologyByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.errorHandler(ctx, http.StatusNotFound, err)
		} else {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	uid, err := authUserIDUint(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}
	if d.CreatorID != uid && !isModeratorFromCtx(ctx) {
		h.errorHandler(ctx, http.StatusForbidden, fmt.Errorf("%w: нет доступа к чужой заявке", repository.ErrNotAllowed))
		return
	}

	constructions, err := h.Repository.GetDendrochronologyConstructionsAPI(d.ID)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	creatorLogin := h.Repository.GetCreatorLogin(d.CreatorID)
	moderatorLogin := h.Repository.GetModeratorLogin(d.ModeratorID)

	var dateFormed, dateCompleted *time.Time
	if d.DateFormed.Valid {
		dateFormed = &d.DateFormed.Time
	}
	if d.DateCompleted.Valid {
		dateCompleted = &d.DateCompleted.Time
	}
	var modLogin *string
	if moderatorLogin != "" {
		modLogin = &moderatorLogin
	}

	ctx.JSON(http.StatusOK, serializer.DendrochronologyDetailJSON{
		ID:             d.ID,
		Status:         d.Status,
		DateCreate:     d.DateCreate,
		DateFormed:     dateFormed,
		DateCompleted:  dateCompleted,
		CreatorLogin:   creatorLogin,
		ModeratorLogin: modLogin,
		TotalSamples:   d.TotalSamples,
		BuildDate:      d.BuildDate,
		Constructions:  constructions,
	})
}

// APIUpdateDendrochronology Обновление полей заявки
// @Summary Обновить заявку
// @Tags dendrochronologies
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Param body body serializer.DendrochronologyUpdateJSON true "Поля для обновления"
// @Security ApiKeyAuth
// @Success 200 {object} serializer.DendrochronologyListJSON
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dendrochronologies/{id} [put]
func (h *Handler) APIUpdateDendrochronology(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный id"))
		return
	}

	var j serializer.DendrochronologyUpdateJSON
	if err := ctx.ShouldBindJSON(&j); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	uid, err := authUserIDUint(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}
	d, err := h.Repository.UpdateDendrochronologyFields(id, j, int(uid))
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

// APIFormDendrochronology Оформить заявку (из черновика)
// @Summary Оформить заявку
// @Tags dendrochronologies
// @Produce json
// @Param id path int true "ID заявки"
// @Security ApiKeyAuth
// @Success 200 {object} serializer.DendrochronologyListJSON
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dendrochronologies/{id}/form [put]
func (h *Handler) APIFormDendrochronology(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный id"))
		return
	}

	uid, err := authUserIDUint(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}
	d, err := h.Repository.FormDendrochronologyAPI(id, int(uid))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.errorHandler(ctx, http.StatusNotFound, err)
		} else if errors.Is(err, repository.ErrNotAllowed) {
			h.errorHandler(ctx, http.StatusForbidden, err)
		} else {
			h.errorHandler(ctx, http.StatusBadRequest, err)
		}
		return
	}

	creatorLogin := h.Repository.GetCreatorLogin(d.CreatorID)
	moderatorLogin := h.Repository.GetModeratorLogin(d.ModeratorID)
	datedCount := h.Repository.GetDatedConstructionsCount(d.ID)
	ctx.JSON(http.StatusOK, serializer.DendrochronologyToListJSON(d, creatorLogin, moderatorLogin, datedCount))
}

// APIFinishDendrochronology Завершить заявку (модератор)
// @Summary Завершить заявку
// @Description Установка итогового статуса; только для модератора.
// @Tags dendrochronologies
// @Accept json
// @Produce json
// @Param id path int true "ID заявки"
// @Param body body serializer.FinishJSON true "Статус завершения"
// @Security ApiKeyAuth
// @Success 200 {object} serializer.DendrochronologyListJSON
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dendrochronologies/{id}/finish [put]
func (h *Handler) APIFinishDendrochronology(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный id"))
		return
	}

	var j serializer.FinishJSON
	if err := ctx.ShouldBindJSON(&j); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	uid, err := authUserIDUint(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}
	d, err := h.Repository.FinishDendrochronologyAPI(id, j.Status, int(uid))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.errorHandler(ctx, http.StatusNotFound, err)
		} else if errors.Is(err, repository.ErrNotAllowed) {
			h.errorHandler(ctx, http.StatusForbidden, err)
		} else {
			h.errorHandler(ctx, http.StatusBadRequest, err)
		}
		return
	}

	creatorLogin := h.Repository.GetCreatorLogin(d.CreatorID)
	moderatorLogin := h.Repository.GetModeratorLogin(d.ModeratorID)
	datedCount := h.Repository.GetDatedConstructionsCount(d.ID)
	ctx.JSON(http.StatusOK, serializer.DendrochronologyToListJSON(d, creatorLogin, moderatorLogin, datedCount))
}

// APIDeleteDendrochronology Удалить заявку
// @Summary Удалить заявку
// @Tags dendrochronologies
// @Produce json
// @Param id path int true "ID заявки"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]string "status: deleted"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /dendrochronologies/{id} [delete]
func (h *Handler) APIDeleteDendrochronology(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный id"))
		return
	}

	uid, err := authUserIDUint(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}
	if err := h.Repository.DeleteDendrochronologyAPI(id, int(uid)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.errorHandler(ctx, http.StatusNotFound, err)
		} else if errors.Is(err, repository.ErrNotAllowed) {
			h.errorHandler(ctx, http.StatusForbidden, err)
		} else {
			h.errorHandler(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
