package ds

import (
	"database/sql"
	"time"
)

const (
	StatusDraft     = "черновик"
	StatusDeleted   = "удалён"
	StatusFormed    = "сформирован"
	StatusCompleted = "завершён"
	StatusRejected  = "отклонён"
)

type Dendrochronology struct {
	ID            uint         `gorm:"primaryKey"`
	Status        string       `gorm:"type:varchar(20);not null"`
	DateCreate    time.Time    `gorm:"not null"`
	DateFormed    sql.NullTime `gorm:"default:null"`
	DateCompleted sql.NullTime `gorm:"default:null"`
	CreatorID     uint         `gorm:"not null"`
	ModeratorID   *uint        `gorm:"default:null"`
	TotalSamples  *int         `gorm:"default:null"`
	BuildDate     *int         `gorm:"column:build_date;default:null"`

	Creator   Users  `gorm:"foreignKey:CreatorID"`
	Moderator *Users `gorm:"foreignKey:ModeratorID"`
}

// TableName — таблица заявок в БД.
func (Dendrochronology) TableName() string {
	return "dendrochronologies"
}
