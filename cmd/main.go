package main

import (
	router "IFJudger/internal"
	"IFJudger/pkg/config"
	"database/sql"
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

func main() {
	envConfigs, err := config.LoadConfig()
	if err != nil {
		panic(err.Error())
	}

	db, err := sql.Open("sqlite", "db.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	log.Println("Database connection success")

	mux := router.StartRoutes(envConfigs, db)
	http.ListenAndServe(":8080", mux)
}
