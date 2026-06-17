package ds

type Construction struct {
	ID                 uint   `gorm:"primaryKey"`
	ConstructionTitle  string `gorm:"type:varchar(100);not null"`
	UseLife            string `gorm:"type:varchar(50);not null"`
	Description        string `gorm:"type:text;not null"`
	ShortDescriptionEn string `gorm:"type:text"`
	ImageURL           string `gorm:"type:varchar(255)"`
	VideoURL           string `gorm:"type:varchar(255)"`
	IsDelete           bool   `gorm:"type:boolean;not null;default:false"`
}
