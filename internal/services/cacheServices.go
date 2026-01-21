package services

import (
	"IFJudger/internal/models"
	"IFJudger/internal/models/configs"
	folderutils "IFJudger/pkg/folder_utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type CacheService struct {
	cacheConfig configs.ConfigCache
}

func StartCacheService(cacheConfig configs.ConfigCache) (*CacheService, error) {
	if err := os.MkdirAll(cacheConfig.CACHEDIRECTORY, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache root: %w", err)
	}
	return &CacheService{
		cacheConfig: cacheConfig,
	}, nil
}

func (s *CacheService) GetProblemData(problemID string) ([]models.LanguageLimits, string, error) {
	problemDir := filepath.Join(s.cacheConfig.CACHEDIRECTORY, problemID+"-problem")

	metaPath := filepath.Join(problemDir, "meta.json")
	_, err := os.Stat(metaPath)
	if os.IsNotExist(err) {
		err = s.downloadAndExtract(problemID, problemDir)
		if err != nil {
			return nil, "", err
		}
	}

	metaFile, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read meta.json: %w", err)
	}

	var limits []models.LanguageLimits
	if err := json.Unmarshal(metaFile, &limits); err != nil {
		return nil, "", fmt.Errorf("corrupted meta.json: %w", err)
	}

	return limits, problemDir, nil
}

func (s *CacheService) downloadAndExtract(problemID string, problemDir string) error {
	fmt.Printf("missing cache, downloading data of %s...\n", problemID)

	apiURL := fmt.Sprintf("%s/%s/package", s.cacheConfig.APIURL, problemID)

	fmt.Println(apiURL)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Admin-Token", s.cacheConfig.APIKEY)
	fmt.Println(req.Header)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %s", resp.Status)
	}

	tmpZip, err := os.CreateTemp("", "problem-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpZip.Name())

	_, err = io.Copy(tmpZip, resp.Body)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	tmpZip.Close()

	return folderutils.Unzip(tmpZip.Name(), problemDir)
}
