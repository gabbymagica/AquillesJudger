package main

import (
	router "IFJudger/internal"
	"net/http"
)

func main() {
	mux := router.StartRoutes()
	http.ListenAndServe(":8080", mux)
}
