package repository

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"

	"metoda/internal/app/ds"
	minio "metoda/internal/app/minioClient"
	"metoda/internal/app/serializer"

	"gorm.io/gorm"
)

func (r *Repository) GetAllConstructions() ([]ds.Construction, error) {
	var constructions []ds.Construction
	err := r.db.Where("is_delete = false").Find(&constructions).Error
	if err != nil {
		return nil, err
	}
	return constructions, nil
}

func (r *Repository) GetConstructionByID(id int) (*ds.Construction, error) {
	var construction ds.Construction
	err := r.db.Where("id = ? AND is_delete = false", id).First(&construction).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: конструкция с id %d", ErrNotFound, id)
		}
		return nil, err
	}
	return &construction, nil
}

func (r *Repository) SearchConstructionsByTitle(title string) ([]ds.Construction, error) {
	var constructions []ds.Construction
	err := r.db.Where("construction_title ILIKE ? AND is_delete = ?", "%"+title+"%", false).Find(&constructions).Error
	if err != nil {
		return nil, err
	}
	return constructions, nil
}

func (r *Repository) CreateConstruction(j serializer.ConstructionJSON) (ds.Construction, error) {
	if j.ConstructionTitle == "" {
		return ds.Construction{}, fmt.Errorf("поле construction_title обязательно")
	}
	if j.UseLife == "" {
		return ds.Construction{}, fmt.Errorf("поле use_life обязательно")
	}
	if j.Description == "" {
		return ds.Construction{}, fmt.Errorf("поле description обязательно")
	}
	c := serializer.ConstructionFromJSON(j)
	if err := r.db.Create(&c).Error; err != nil {
		return ds.Construction{}, err
	}
	return c, nil
}

func (r *Repository) UploadConstructionImage(ctx context.Context, constructionID int, file *multipart.FileHeader) (ds.Construction, error) {
	c, err := r.GetConstructionByID(constructionID)
	if err != nil {
		return ds.Construction{}, err
	}
	if err := minio.EnsureBucket(ctx, r.mc, minio.ConstructionsBucket); err != nil {
		return ds.Construction{}, err
	}
	objectName, err := minio.UploadFile(ctx, r.mc, minio.ConstructionsBucket, file)
	if err != nil {
		return ds.Construction{}, err
	}
	c.ImageURL = objectName
	if err := r.db.Save(c).Error; err != nil {
		return ds.Construction{}, err
	}
	return *c, nil
}

func (r *Repository) UploadConstructionVideo(ctx context.Context, constructionID int, file *multipart.FileHeader) (ds.Construction, error) {
	c, err := r.GetConstructionByID(constructionID)
	if err != nil {
		return ds.Construction{}, err
	}
	if err := minio.EnsureBucket(ctx, r.mc, minio.ConstructionsBucket); err != nil {
		return ds.Construction{}, err
	}
	objectName, err := minio.UploadFile(ctx, r.mc, minio.ConstructionsBucket, file)
	if err != nil {
		return ds.Construction{}, err
	}
	c.VideoURL = objectName
	if err := r.db.Save(c).Error; err != nil {
		return ds.Construction{}, err
	}
	return *c, nil
}
