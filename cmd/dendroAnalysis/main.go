package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	_ "metoda/docs"
	"metoda/internal/app/config"
	"metoda/internal/app/dsn"
	"metoda/internal/app/handler"
	"metoda/internal/app/repository"
	"metoda/internal/pkg"
)

// @title Dendrochronology API
// @version 1.0
// @description REST API
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	router := gin.Default()

	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	postgresString := dsn.FromEnv()
	fmt.Println(postgresString)

	rep, errRep := repository.New(postgresString)
	if errRep != nil {
		logrus.Fatalf("error initializing repository: %v", errRep)
	}

	hand := handler.NewHandler(rep)

	dendrochronology := pkg.NewApp(conf, router, hand)
	dendrochronology.RunApp()
}
