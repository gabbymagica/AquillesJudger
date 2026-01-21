package main

import (
	router "IFJudger/internal"
	"IFJudger/pkg/config"
	"net/http"
)

func main() {
	envConfigs, err := config.LoadConfig()
	if err != nil {
		panic(err.Error())
	}

	mux := router.StartRoutes(envConfigs)
	http.ListenAndServe(":8080", mux)
}
