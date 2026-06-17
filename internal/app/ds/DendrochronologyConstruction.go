package ds

type DendrochronologyConstruction struct {
	ID                 uint   `gorm:"primaryKey"`
	DendrochronologyID uint   `gorm:"not null;uniqueIndex:idx_dendro_construction"`
	ConstructionID     uint   `gorm:"not null;uniqueIndex:idx_dendro_construction"`
	SamplesCount       int    `gorm:"not null;default:1"`
	CuttingDate        string `gorm:"type:varchar(20)"`
	DateCorrection     string `gorm:"type:varchar(20)"`
	// UseLifeOverride — значение именно у этой строки заявки; пусто — брать Construction.UseLife из справочника.
	UseLifeOverride string `gorm:"type:varchar(255);column:use_life_override"`

	Dendrochronology Dendrochronology `gorm:"foreignKey:DendrochronologyID"`
	Construction    Construction     `gorm:"foreignKey:ConstructionID"`
}

// TableName — таблица связи заявка–конструкция в БД.
func (DendrochronologyConstruction) TableName() string {
	return "dendrochronology_constructions"
}
