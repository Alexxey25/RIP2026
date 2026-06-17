package serializer

import "metoda/internal/app/ds"

type ConstructionJSON struct {
	ID                 uint   `json:"id"`
	ConstructionTitle  string `json:"construction_title"`
	UseLife            string `json:"use_life"`
	Description        string `json:"description"`
	ShortDescriptionEn string `json:"short_description_en"`
	ImageURL           string `json:"image_url"`
	VideoURL           string `json:"video_url"`
	IsDelete           bool   `json:"is_delete"`
}

func ConstructionToJSON(c ds.Construction) ConstructionJSON {
	return ConstructionJSON{
		ID:                 c.ID,
		ConstructionTitle:  c.ConstructionTitle,
		UseLife:            c.UseLife,
		Description:        c.Description,
		ShortDescriptionEn: c.ShortDescriptionEn,
		ImageURL:           c.ImageURL,
		VideoURL:           c.VideoURL,
		IsDelete:           c.IsDelete,
	}
}

func ConstructionFromJSON(j ConstructionJSON) ds.Construction {
	return ds.Construction{
		ConstructionTitle:  j.ConstructionTitle,
		UseLife:            j.UseLife,
		Description:        j.Description,
		ShortDescriptionEn: j.ShortDescriptionEn,
		ImageURL:           j.ImageURL,
		VideoURL:           j.VideoURL,
		IsDelete:           j.IsDelete,
	}
}
