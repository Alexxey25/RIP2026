package serializer

// DendroConstructionUpdateJSON — editable fields in a dendrochronology-construction link.
type DendroConstructionUpdateJSON struct {
	SamplesCount   int    `json:"samples_count"`
	CuttingDate    string `json:"cutting_date"`
	DateCorrection string `json:"date_correction"`
	UseLife        string `json:"use_life"`
}
