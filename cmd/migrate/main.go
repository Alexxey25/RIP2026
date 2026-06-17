package main

import (
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"metoda/internal/app/ds"
	"metoda/internal/app/dsn"
)

func main() {
	_ = godotenv.Load()
	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.AutoMigrate(
		&ds.Users{},
		&ds.Construction{},
		&ds.Dendrochronology{},
		&ds.DendrochronologyConstruction{},
	)
	if err != nil {
		panic("cant migrate db")
	}
}
