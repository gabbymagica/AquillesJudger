package main

import (
	"IFJudger/internal/models/configs"
	"IFJudger/internal/services"
	"fmt"
)

func main() {
	cacheService, err := services.StartCacheService(configs.ConfigCache{
		APIURL:             "http://localhost:55555/CasoTeste/problemaInterno",
		APIKEY:             "token-mega-secreto-que-ninguem-nunca-sabera-#trocarissodepoispraacessardoenv",
		CACHEDIRECTORY:     "../internal/api/cache",
		CACHEFILEEXTENSION: "-cases.cache"})
	if err != nil {
		fmt.Println(err)
	}

	usecases, err := cacheService.GetUseCases(7)
	fmt.Println(usecases)
	fmt.Println(err)

	/*mux := router.StartRoutes()
	http.ListenAndServe(":8080", mux)*/
}
