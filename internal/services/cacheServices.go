package services

import (
	"IFJudger/internal/models"
	"IFJudger/internal/models/configs"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type CacheService struct {
	cacheConfig configs.ConfigCache
}

var ErrCacheMiss = errors.New("cache not found")
var ErrCacheInvalid = errors.New("invalid cache")

func StartCacheService(cacheConfig configs.ConfigCache) (*CacheService, error) {
	return &CacheService{
		cacheConfig: cacheConfig,
	}, nil
}

func (s *CacheService) GetUseCases(problemID int) ([]models.UseCases, error) {

	fileName := fmt.Sprintf("%d%s.json", problemID, s.cacheConfig.CACHEFILEEXTENSION)
	fullPath := filepath.Join(s.cacheConfig.CACHEDIRECTORY, fileName)

	cachedCases, err := s.readCache(fullPath)
	if err == nil {
		return cachedCases, nil
	}

	apiCases, bodyBytes, err := s.requestUseCase(problemID)
	if err != nil {
		return nil, fmt.Errorf("error while trying to obtain useCases: %w", err)
	}

	if err := s.writeCache(fullPath, bodyBytes); err != nil {
		fmt.Printf("Warning: failed to save cache for %d: %v\n", problemID, err)
	}

	return apiCases, nil
}

func (s *CacheService) writeCache(filePath string, body []byte) error {
	if err := os.MkdirAll(s.cacheConfig.CACHEDIRECTORY, 0755); err != nil {
		fmt.Println("Error while trying to make Directory", err)
		return nil
	}

	if err := os.WriteFile(filePath, body, 0644); err != nil {
		fmt.Println("Error trying to save cache:", err)
	}

	return nil
}

func (s *CacheService) requestUseCase(problemID int) ([]models.UseCases, []byte, error) {
	apiURL := fmt.Sprintf("%s/%d", s.cacheConfig.APIURL, problemID)
	fmt.Println(apiURL)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed at creating request: %w", err)
	}

	req.Header.Set("X-Admin-Token", s.cacheConfig.APIKEY)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request falhou: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API returned status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("body reading failed: %w", err)
	}

	fmt.Println(string(body))
	var container models.ProblemUseCases
	if err := json.Unmarshal(body, &container); err != nil {
		return nil, nil, fmt.Errorf("api decodification failed: %w", err)
	}

	return container.UseCases, body, nil
}

func (s *CacheService) readCache(path string) ([]models.UseCases, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	var container models.ProblemUseCases
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, fmt.Errorf("corrupted JSON: %w", err)
	}

	return container.UseCases, nil
}
