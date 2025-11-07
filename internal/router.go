package router

import (
	"IFJudger/internal/controllers"
	"IFJudger/internal/services"
	"fmt"
	"net/http"
)

func StartRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	testController, err := controllers.StartTestController()
	if err != nil {
		fmt.Println(err.Error())
	}

	// inicializa os serviços
	workerServices, err := services.StartWorkerService()
	if err != nil {
		fmt.Println(err.Error())
	}

	workerController, err := controllers.StartWorkerController(workerServices)
	if err != nil {
		panic(err)
	}

	mux.HandleFunc("GET /test", testController.GetTest)
	mux.HandleFunc("POST /worker", workerController.HandleExecution)

	return mux
}
