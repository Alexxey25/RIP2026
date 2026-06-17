package repository

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"metoda/internal/app/ds"
	"metoda/internal/app/serializer"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func (r *Repository) ClearDraftDendrochronologiesOnStartup() {
	r.db.Exec("UPDATE dendrochronologies SET status = ? WHERE status = ?", ds.StatusDeleted, ds.StatusDraft)
}

func (r *Repository) GetCreatorLogin(creatorID uint) string {
	var u ds.Users
	r.db.Where("id = ?", creatorID).First(&u)
	return u.Login
}

func (r *Repository) GetModeratorLogin(moderatorID *uint) string {
	if moderatorID == nil {
		return ""
	}
	var u ds.Users
	r.db.Where("id = ?", *moderatorID).First(&u)
	return u.Login
}

// GetDatedConstructionsCount — число записей м-м, в которых рассчитываемое поле результата (год по дате рубки + поправка) не пустое.
func (r *Repository) GetDatedConstructionsCount(dendrochronologyID uint) int {
	var count int64
	r.db.Model(&ds.DendrochronologyConstruction{}).
		Where("dendrochronology_id = ?", dendrochronologyID).
		Where("trim(COALESCE(cutting_date,'')) != '' AND trim(COALESCE(date_correction,'')) != ''").
		Count(&count)
	return int(count)
}

// ─── HTML-layer methods (kept for lab2 compatibility) ───────────────────────

func (r *Repository) GetDraftDendrochronology(creatorID uint) (*ds.Dendrochronology, error) {
	var d ds.Dendrochronology
	err := r.db.Where("creator_id = ? AND status = ?", creatorID, ds.StatusDraft).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) GetDendrochronologyWithConstructions(dendrochronologyID uint) (*ds.Dendrochronology, []DendrochronologyConstructionView, error) {
	var d ds.Dendrochronology
	err := r.db.First(&d, dendrochronologyID).Error
	if err != nil {
		return nil, nil, err
	}
	if d.Status == ds.StatusDeleted {
		return nil, nil, nil
	}
	var items []ds.DendrochronologyConstruction
	err = r.db.Where("dendrochronology_id = ?", dendrochronologyID).Preload("Construction").Order("id").Find(&items).Error
	if err != nil {
		return nil, nil, err
	}
	var views []DendrochronologyConstructionView
	for _, item := range items {
		life := strings.TrimSpace(item.UseLifeOverride)
		if life == "" {
			life = item.Construction.UseLife
		}
		views = append(views, DendrochronologyConstructionView{
			ID:                item.ID,
			ConstructionTitle: item.Construction.ConstructionTitle,
			UseLife:           life,
			ImageURL:          item.Construction.ImageURL,
			SamplesCount:      item.SamplesCount,
			CuttingDate:       item.CuttingDate,
			DateCorrection:    item.DateCorrection,
		})
	}
	return &d, views, nil
}

type DendrochronologyConstructionView struct {
	ID                uint
	ConstructionTitle string
	UseLife           string
	ImageURL          string
	SamplesCount      int
	CuttingDate       string
	DateCorrection    string
}

func (r *Repository) GetEstimatedBuildYear(dendrochronologyID uint) int {
	var year int
	err := r.db.Raw(`
		SELECT COALESCE(MAX(cutting_date::int + date_correction::int), 0)
		FROM dendrochronology_constructions
		WHERE dendrochronology_id = ?
		  AND cutting_date  IS NOT NULL AND trim(cutting_date)  != ''
		  AND date_correction IS NOT NULL AND trim(date_correction) != ''
	`, dendrochronologyID).Scan(&year).Error
	if err != nil {
		logrus.Errorf("GetEstimatedBuildYear: %v", err)
	}
	return year
}

func (r *Repository) AddConstructionToDendrochronology(constructionID uint, creatorID uint) error {
	var d ds.Dendrochronology
	err := r.db.Where("creator_id = ? AND status = ?", creatorID, ds.StatusDraft).First(&d).Error
	if err != nil {
		d = ds.Dendrochronology{
			Status:     ds.StatusDraft,
			DateCreate: time.Now(),
			CreatorID:  creatorID,
		}
		if err := r.db.Create(&d).Error; err != nil {
			return err
		}
	}
	dc := ds.DendrochronologyConstruction{
		DendrochronologyID: d.ID,
		ConstructionID:     constructionID,
		SamplesCount:       1,
	}
	return r.db.Create(&dc).Error
}

func (r *Repository) DeleteDendrochronologyBySQL(dendrochronologyID uint) error {
	return r.db.Exec("UPDATE dendrochronologies SET status = '"+ds.StatusDeleted+"' WHERE id = ?", dendrochronologyID).Error
}

// UpdateDendrochronologyItem обновляет количество образцов и поля дат
// для одной строки связи заявки с конструкцией (HTML-форма на странице корзины).
func (r *Repository) UpdateDendrochronologyItem(itemID uint, samplesCount int, cuttingDate, dateCorrection string) error {
	var item ds.DendrochronologyConstruction
	if err := r.db.First(&item, itemID).Error; err != nil {
		return err
	}

	if samplesCount < 1 {
		samplesCount = 1
	}

	updates := map[string]interface{}{
		"samples_count":   samplesCount,
		"cutting_date":    cuttingDate,
		"date_correction": dateCorrection,
	}

	return r.db.Model(&item).Updates(updates).Error
}

func (r *Repository) FormDendrochronology(dendrochronologyID uint) error {
	totalSamples := r.GetTotalSamples(dendrochronologyID)
	buildYear := r.GetEstimatedBuildYear(dendrochronologyID)
	now := time.Now()
	updates := map[string]interface{}{
		"status":        ds.StatusFormed,
		"date_formed":   now,
		"total_samples": totalSamples,
	}
	if buildYear > 0 {
		updates["build_date"] = buildYear
	}
	return r.db.Model(&ds.Dendrochronology{}).Where("id = ?", dendrochronologyID).Updates(updates).Error
}

func (r *Repository) GetTotalSamples(dendrochronologyID uint) int {
	var total int
	r.db.Model(&ds.DendrochronologyConstruction{}).
		Where("dendrochronology_id = ?", dendrochronologyID).
		Select("COALESCE(SUM(samples_count), 0)").
		Scan(&total)
	return total
}

func (r *Repository) GetCartCount(creatorID uint) int64 {
	var dendrochronologyID uint
	var count int64
	err := r.db.Model(&ds.Dendrochronology{}).Where("creator_id = ? AND status = ?", creatorID, ds.StatusDraft).Select("id").First(&dendrochronologyID).Error
	if err != nil {
		return 0
	}
	err = r.db.Model(&ds.DendrochronologyConstruction{}).Where("dendrochronology_id = ?", dendrochronologyID).Count(&count).Error
	if err != nil {
		logrus.Println("Error counting records:", err)
	}
	return count
}

func (r *Repository) GetDraftDendrochronologyID(creatorID uint) uint {
	var dendrochronologyID uint
	err := r.db.Model(&ds.Dendrochronology{}).Where("creator_id = ? AND status = ?", creatorID, ds.StatusDraft).Select("id").First(&dendrochronologyID).Error
	if err != nil {
		return 0
	}
	return dendrochronologyID
}

// ─── API methods ─────────────────────────────────────────────────────────────

// GetDendrochronologyByID returns a non-deleted dendrochronology by ID.
func (r *Repository) GetDendrochronologyByID(id int) (ds.Dendrochronology, error) {
	var d ds.Dendrochronology
	err := r.db.Where("id = ?", id).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.Dendrochronology{}, fmt.Errorf("%w: заявка с id %d", ErrNotFound, id)
		}
		return ds.Dendrochronology{}, err
	}
	if d.Status == ds.StatusDeleted {
		return ds.Dendrochronology{}, fmt.Errorf("%w: заявка удалена", ErrNotFound)
	}
	return d, nil
}

// GetAllDendrochronologies возвращает заявки кроме удалённых и черновика, с фильтром по диапазону даты формирования и статусу.
// Если isModerator == false, возвращаются только заявки с creator_id = viewerID.
func (r *Repository) GetAllDendrochronologies(from, to time.Time, status string, viewerID uint, isModerator bool, limit, offset int) ([]ds.Dendrochronology, int64, error) {
	var list []ds.Dendrochronology
	var total int64

	sub := r.db.Model(&ds.Dendrochronology{}).Where("status != ? AND status != ?", ds.StatusDeleted, ds.StatusDraft)
	if !isModerator {
		sub = sub.Where("creator_id = ?", viewerID)
	}
	if !from.IsZero() {
		sub = sub.Where("date_formed >= ?", from)
	}
	if !to.IsZero() {
		sub = sub.Where("date_formed <= ?", to.Add(24*time.Hour))
	}
	if status != "" && status != "Все" {
		sub = sub.Where("status = ?", status)
	}
	
	sub.Count(&total)

	err := sub.Order("id").Limit(limit).Offset(offset).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetDendrochronologyConstructionsAPI returns the m-m records for a dendrochronology with construction data.
func (r *Repository) GetDendrochronologyConstructionsAPI(id uint) ([]serializer.DendroConstructionViewJSON, error) {
	var items []ds.DendrochronologyConstruction
	err := r.db.Where("dendrochronology_id = ?", id).Preload("Construction").Order("id").Find(&items).Error
	if err != nil {
		return nil, err
	}
	views := make([]serializer.DendroConstructionViewJSON, 0, len(items))
	for _, item := range items {
		life := strings.TrimSpace(item.UseLifeOverride)
		if life == "" {
			life = item.Construction.UseLife
		}
		views = append(views, serializer.DendroConstructionViewJSON{
			ID:                item.ID,
			ConstructionID:    item.ConstructionID,
			ConstructionTitle: item.Construction.ConstructionTitle,
			UseLife:           life,
			ImageURL:          item.Construction.ImageURL,
			SamplesCount:      item.SamplesCount,
			CuttingDate:       item.CuttingDate,
			DateCorrection:    item.DateCorrection,
		})
	}
	return views, nil
}

// GetCartInfo returns the draft dendrochronology ID and constructions count for the given user.
func (r *Repository) GetCartInfo(creatorID uint) (uint, int64, error) {
	var d ds.Dendrochronology
	err := r.db.Where("creator_id = ? AND status = ?", creatorID, ds.StatusDraft).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	var count int64
	r.db.Model(&ds.DendrochronologyConstruction{}).Where("dendrochronology_id = ?", d.ID).Count(&count)
	return d.ID, count, nil
}

// UpdateDendrochronologyFields обновляет редактируемые поля заявки (например build_date). Только для черновика и только создатель.
func (r *Repository) UpdateDendrochronologyFields(id int, j serializer.DendrochronologyUpdateJSON, currentUserID int) (ds.Dendrochronology, error) {
	d, err := r.GetDendrochronologyByID(id)
	if err != nil {
		return ds.Dendrochronology{}, err
	}
	if d.Status != ds.StatusDraft {
		return ds.Dendrochronology{}, fmt.Errorf("%w: можно менять только черновик", ErrNotAllowed)
	}
	if int(d.CreatorID) != currentUserID {
		return ds.Dendrochronology{}, fmt.Errorf("%w: только создатель может редактировать заявку", ErrNotAllowed)
	}
	if j.BuildDate == nil {
		return d, nil
	}
	// Явный UPDATE: GORM Updates(map) иногда не обновляет колонку в PostgreSQL
	if err := r.db.Exec("UPDATE dendrochronologies SET build_date = ? WHERE id = ?", *j.BuildDate, id).Error; err != nil {
		return ds.Dendrochronology{}, err
	}
	d.BuildDate = j.BuildDate
	return d, nil
}

// FormDendrochronologyAPI sets status to "сформирован", calculates and stores build_year and total_samples.
func (r *Repository) FormDendrochronologyAPI(id int, currentUserID int) (ds.Dendrochronology, error) {
	d, err := r.GetDendrochronologyByID(id)
	if err != nil {
		return ds.Dendrochronology{}, err
	}
	if d.Status != ds.StatusDraft {
		return ds.Dendrochronology{}, fmt.Errorf("нельзя сформировать заявку со статусом %s", d.Status)
	}

	// Check creator permission
	if int(d.CreatorID) != currentUserID {
		return ds.Dendrochronology{}, fmt.Errorf("%w: только создатель может сформировать заявку", ErrNotAllowed)
	}

	// Mandatory fields check: application must have at least one construction
	var count int64
	r.db.Model(&ds.DendrochronologyConstruction{}).Where("dendrochronology_id = ?", d.ID).Count(&count)
	if count == 0 {
		return ds.Dendrochronology{}, fmt.Errorf("нельзя сформировать пустую заявку: добавьте хотя бы одну конструкцию")
	}

	// Calculate total_samples and build_year
	totalSamples := r.GetTotalSamples(d.ID)
	buildYear := r.GetEstimatedBuildYear(d.ID)
	now := time.Now()

	updates := map[string]interface{}{
		"status":        ds.StatusFormed,
		"date_formed":   now,
		"total_samples": totalSamples,
	}
	if buildYear > 0 {
		updates["build_date"] = buildYear
	}

	if err := r.db.Model(&d).Updates(updates).Error; err != nil {
		return ds.Dendrochronology{}, err
	}

	// Re-fetch to get updated values
	return r.GetDendrochronologyByID(id)
}

// FinishDendrochronologyAPI lets a moderator set status to "завершён" or "отклонён".
func (r *Repository) FinishDendrochronologyAPI(id int, status string, currentUserID int) (ds.Dendrochronology, error) {
	if status != ds.StatusCompleted && status != ds.StatusRejected {
		return ds.Dendrochronology{}, fmt.Errorf("недопустимый статус: ожидается '%s' или '%s'", ds.StatusCompleted, ds.StatusRejected)
	}

	moderator, err := r.GetUserByID(currentUserID)
	if err != nil {
		return ds.Dendrochronology{}, err
	}
	if !moderator.IsModerator {
		return ds.Dendrochronology{}, fmt.Errorf("%w: только модератор может завершить или отклонить заявку", ErrNotAllowed)
	}

	d, err := r.GetDendrochronologyByID(id)
	if err != nil {
		return ds.Dendrochronology{}, err
	}
	if d.Status != ds.StatusFormed {
		return ds.Dendrochronology{}, fmt.Errorf("завершить или отклонить можно только сформированную заявку; текущий статус — %s (сначала PUT .../form)", d.Status)
	}

	now := time.Now()
	moderatorID := uint(moderator.ID)
	if err := r.db.Model(&d).Updates(map[string]interface{}{
		"status":         status,
		"date_completed": now,
		"moderator_id":   moderatorID,
	}).Error; err != nil {
		return ds.Dendrochronology{}, err
	}

	return r.GetDendrochronologyByID(id)
}

// DeleteDendrochronologyAPI soft-deletes a draft dendrochronology (creator only).
func (r *Repository) DeleteDendrochronologyAPI(id int, currentUserID int) error {
	d, err := r.GetDendrochronologyByID(id)
	if err != nil {
		return err
	}
	if d.Status != ds.StatusDraft {
		return fmt.Errorf("%w: только черновик может быть удалён создателем", ErrNotAllowed)
	}
	if int(d.CreatorID) != currentUserID {
		return fmt.Errorf("%w: только создатель может удалить заявку", ErrNotAllowed)
	}
	return r.db.Model(&d).Update("status", ds.StatusDeleted).Error
}

// AddConstructionToCartAPI adds a construction to the user's draft (creates draft if needed).
func (r *Repository) AddConstructionToCartAPI(constructionID uint, creatorID uint) (ds.Dendrochronology, bool, error) {
	// Verify construction exists and is not deleted
	var c ds.Construction
	if err := r.db.Where("id = ? AND is_delete = false", constructionID).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.Dendrochronology{}, false, fmt.Errorf("%w: конструкция с id %d", ErrNotFound, constructionID)
		}
		return ds.Dendrochronology{}, false, err
	}

	// Get or create draft
	var d ds.Dendrochronology
	created := false
	err := r.db.Where("creator_id = ? AND status = ?", creatorID, ds.StatusDraft).First(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		d = ds.Dendrochronology{
			Status:     ds.StatusDraft,
			DateCreate: time.Now(),
			CreatorID:  creatorID,
		}
		if err := r.db.Create(&d).Error; err != nil {
			return ds.Dendrochronology{}, false, err
		}
		created = true
	} else if err != nil {
		return ds.Dendrochronology{}, false, err
	}

	// Check for duplicate
	var existing ds.DendrochronologyConstruction
	res := r.db.Where("dendrochronology_id = ? AND construction_id = ?", d.ID, constructionID).First(&existing)
	if res.Error == nil {
		return ds.Dendrochronology{}, false, fmt.Errorf("%w: конструкция %d уже добавлена в заявку %d", ErrAlreadyExists, constructionID, d.ID)
	}

	dc := ds.DendrochronologyConstruction{
		DendrochronologyID: d.ID,
		ConstructionID:     constructionID,
		SamplesCount:       1,
	}
	if err := r.db.Create(&dc).Error; err != nil {
		return ds.Dendrochronology{}, false, err
	}
	return d, created, nil
}

// DeleteConstructionFromCartAPI removes a construction from a dendrochronology (must be draft).
func (r *Repository) DeleteConstructionFromCartAPI(constructionID, dendrochronologyID int, currentUserID int) (ds.Dendrochronology, error) {
	d, err := r.GetDendrochronologyByID(dendrochronologyID)
	if err != nil {
		return ds.Dendrochronology{}, err
	}
	if int(d.CreatorID) != currentUserID {
		return ds.Dendrochronology{}, fmt.Errorf("%w: только создатель может изменять заявку", ErrNotAllowed)
	}
	if d.Status != ds.StatusDraft {
		return ds.Dendrochronology{}, fmt.Errorf("%w: нельзя изменить не черновик", ErrNotAllowed)
	}
	err = r.db.Where("construction_id = ? AND dendrochronology_id = ?", constructionID, dendrochronologyID).Delete(&ds.DendrochronologyConstruction{}).Error
	if err != nil {
		return ds.Dendrochronology{}, err
	}
	return d, nil
}

// UpdateConstructionInCartAPI edits samples_count, cutting_date, date_correction in m-m.
func (r *Repository) UpdateConstructionInCartAPI(constructionID, dendrochronologyID int, j serializer.DendroConstructionUpdateJSON, currentUserID int) (ds.DendrochronologyConstruction, error) {
	d, err := r.GetDendrochronologyByID(dendrochronologyID)
	if err != nil {
		return ds.DendrochronologyConstruction{}, err
	}
	if int(d.CreatorID) != currentUserID {
		return ds.DendrochronologyConstruction{}, fmt.Errorf("%w: только создатель может изменять заявку", ErrNotAllowed)
	}
	if d.Status != ds.StatusDraft {
		return ds.DendrochronologyConstruction{}, fmt.Errorf("%w: нельзя изменить не черновик", ErrNotAllowed)
	}

	var item ds.DendrochronologyConstruction
	err = r.db.Where("construction_id = ? AND dendrochronology_id = ?", constructionID, dendrochronologyID).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ds.DendrochronologyConstruction{}, fmt.Errorf("%w: связь не найдена", ErrNotFound)
		}
		return ds.DendrochronologyConstruction{}, err
	}

	updates := map[string]interface{}{}
	if j.SamplesCount > 0 {
		updates["samples_count"] = j.SamplesCount
	}
	updates["cutting_date"] = j.CuttingDate
	updates["date_correction"] = j.DateCorrection
	updates["use_life_override"] = j.UseLife

	if err := r.db.Model(&item).Updates(updates).Error; err != nil {
		return ds.DendrochronologyConstruction{}, err
	}

	if err := r.db.Preload("Construction").First(&item, item.ID).Error; err != nil {
		return ds.DendrochronologyConstruction{}, err
	}
	return item, nil
}
