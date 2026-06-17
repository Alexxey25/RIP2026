package serializer

import (
	"metoda/internal/app/ds"
	"time"
)

// DendrochronologyListJSON — used for the list endpoint.
type DendrochronologyListJSON struct {
	ID                      uint       `json:"id"`
	Status                  string     `json:"status"`
	DateCreate              time.Time  `json:"date_create"`
	DateFormed              *time.Time `json:"date_formed"`
	DateCompleted           *time.Time `json:"date_completed"`
	CreatorLogin            string     `json:"creator_login"`
	ModeratorLogin          *string    `json:"moderator_login"`
	TotalSamples            *int       `json:"total_samples"`
	BuildDate               *int       `json:"build_date"`
	DatedConstructionsCount int        `json:"dated_constructions_count"`
}

// DendrochronologyDetailJSON — used for the single-record endpoint (includes constructions).
type DendrochronologyDetailJSON struct {
	ID             uint                       `json:"id"`
	Status         string                     `json:"status"`
	DateCreate     time.Time                  `json:"date_create"`
	DateFormed     *time.Time                 `json:"date_formed"`
	DateCompleted  *time.Time                 `json:"date_completed"`
	CreatorLogin   string                     `json:"creator_login"`
	ModeratorLogin *string                    `json:"moderator_login"`
	TotalSamples   *int                       `json:"total_samples"`
	BuildDate      *int                       `json:"build_date"`
	Constructions  []DendroConstructionViewJSON `json:"constructions"`
}

// DendroConstructionViewJSON — one construction inside a dendrochronology detail.
type DendroConstructionViewJSON struct {
	ID                uint   `json:"id"`
	ConstructionID    uint   `json:"construction_id"`
	ConstructionTitle string `json:"construction_title"`
	UseLife           string `json:"use_life"`
	ImageURL          string `json:"image_url"`
	SamplesCount      int    `json:"samples_count"`
	CuttingDate       string `json:"cutting_date"`
	DateCorrection    string `json:"date_correction"`
}

// DendrochronologyUpdateJSON — поля, которые создатель может менять через PUT (черновик).
type DendrochronologyUpdateJSON struct {
	BuildDate *int `json:"build_date"`
}

// FinishJSON — status sent by moderator to finish or reject.
type FinishJSON struct {
	Status string `json:"status"`
}

// CartJSON — cart icon response.
type CartJSON struct {
	DendrochronologyID uint  `json:"dendrochronology_id"`
	ConstructionsCount int64 `json:"constructions_count"`
}

func DendrochronologyToListJSON(d ds.Dendrochronology, creatorLogin string, moderatorLogin string, datedCount int) DendrochronologyListJSON {
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
	return DendrochronologyListJSON{
		ID:                      d.ID,
		Status:                  d.Status,
		DateCreate:              d.DateCreate,
		DateFormed:              dateFormed,
		DateCompleted:           dateCompleted,
		CreatorLogin:            creatorLogin,
		ModeratorLogin:          modLogin,
		TotalSamples:            d.TotalSamples,
		BuildDate:               d.BuildDate,
		DatedConstructionsCount: datedCount,
	}
}
