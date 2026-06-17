package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/go-faker/faker/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"metoda/internal/app/ds"
	"metoda/internal/app/dsn"
)

func main() {
	postgresString := dsn.FromEnv()
	if postgresString == "" {
		log.Fatal("Empty DSN from env. Set DB_HOST etc.")
	}

	db, err := gorm.Open(postgres.Open(postgresString), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Seeding database...")

	// Make sure we have some users
	var users []ds.Users
	db.Find(&users)
	if len(users) == 0 {
		users = []ds.Users{
			{Login: "user10", Password: "password"},
			{Login: "user11", Password: "password"},
			{Login: "user12", Password: "password"},
		}
		if err := db.Create(&users).Error; err != nil {
			log.Fatalf("Could not create users: %v", err)
		}
	}

	statuses := []string{ds.StatusFormed, ds.StatusCompleted, ds.StatusRejected, ds.StatusDeleted}
	numApplications := 100000
	
	// Create batches for faster insert
	batchSize := 1000
	var batch []ds.Dendrochronology

	for i := 0; i < numApplications; i++ {
		creator := users[rand.Intn(len(users))]
		
		status := statuses[rand.Intn(len(statuses))]
		dateCreate := time.Now().AddDate(0, -rand.Intn(12), -rand.Intn(28))
		
		app := ds.Dendrochronology{
			Status:     status,
			CreatorID:  creator.ID,
			DateCreate: dateCreate,
		}
		if status != ds.StatusDraft && status != ds.StatusDeleted {
			app.DateFormed = sql.NullTime{Time: dateCreate, Valid: true}
		}

		faker.FakeData(&app.BuildDate)
		
		batch = append(batch, app)

		if len(batch) >= batchSize {
			if err := db.Create(&batch).Error; err != nil {
				log.Printf("could not create application batch: %v", err)
			}
			batch = nil
			if (i+1)%10000 == 0 {
				fmt.Printf("Created %d/%d applications\n", i+1, numApplications)
			}
		}
	}
	
	if len(batch) > 0 {
		if err := db.Create(&batch).Error; err != nil {
			log.Printf("could not create application batch: %v", err)
		}
	}

	log.Println("Seeding completed successfully!")
}
