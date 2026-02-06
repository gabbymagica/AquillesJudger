package router

import (
	"IFJudger/internal/controllers"
	"IFJudger/internal/models/configs"
	"IFJudger/internal/repository"
	"IFJudger/internal/services"
	"IFJudger/pkg/config"
	"database/sql"
	"net/http"
)

func StartRoutes(config *config.Config, db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()

	testController, err := controllers.StartTestController()
	if err != nil {
		panic(err.Error())
	}

	cacheService, err := services.StartCacheService(configs.ConfigCache{
		ONLYLOCAL:          config.OnlyLocalCache,
		APIURL:             config.APIUrl,
		APIKEY:             config.APIKey,
		CACHEDIRECTORY:     config.CacheDirectory,
		CACHEFILEEXTENSION: config.CacheFileExtension,
	})
	if err != nil {
		panic(err.Error())
	}

	submissionRepository, err := repository.StartSubmissionRepository(db)
	if err != nil {
		panic(err.Error())
	}

	workerService, err := services.StartWorkerService(configs.WorkerServiceConfig{
		ExecutionDirectory: config.ExecutionDirectory,
		CallbackUrl:        config.CallbackUrl,
		RunnerPath:         config.RunnerBinaryPath,
		ContainerTimeout:   config.ContainerTimeout,
		MaxWorkers:         config.MaxWorkers,
		QueueSize:          config.QueueSize,
	}, submissionRepository)
	if err != nil {
		panic(err.Error())
	}

	judgerService, err := services.StartJudgerService(workerService, cacheService)
	if err != nil {
		panic(err.Error())
	}

	judgerController, err := controllers.StartJudgerController(judgerService)
	if err != nil {
		panic(err.Error())
	}

	mux.HandleFunc("GET /test", testController.GetTest)
	mux.HandleFunc("POST /submit", judgerController.HandleSubmission)
	mux.HandleFunc("GET /job", judgerController.HandleStatus)

	return mux
}
