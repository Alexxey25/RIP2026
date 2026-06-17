package repository

import (
	"errors"
	"fmt"

	"github.com/minio/minio-go/v7"
	minioClient "metoda/internal/app/minioClient"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"metoda/internal/app/ds"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrNotAllowed   = errors.New("not allowed")
	ErrNoDraft      = errors.New("no draft for this user")
)

type Repository struct {
	db     *gorm.DB
	mc     *minio.Client
	rd     *redis.Client // Redis: blacklist JWT; nil если не подключён
	userID int
}

func New(dsn string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&ds.Users{},
		&ds.Construction{},
		&ds.Dendrochronology{},
		&ds.DendrochronologyConstruction{},
	); err != nil {
		return nil, fmt.Errorf("db automigrate: %w", err)
	}

	mc, err := minioClient.InitMinio()
	if err != nil {
		return nil, err
	}

	rdb, err := connectRedis()
	if err != nil {
		return nil, fmt.Errorf("redis: %w", err)
	}

	return &Repository{
		db:     db,
		mc:     mc,
		rd:     rdb,
		userID: 1,
	}, nil
}

func (r *Repository) GetUserID() int {
	return r.userID
}

func (r *Repository) SetUserID(id int) {
	r.userID = id
}

func (r *Repository) SignOut() {
	r.userID = 0
}
