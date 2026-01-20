package worker

import (
	folderutils "IFJudger/pkg/folder_utils"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type LanguageID int

const (
	Python LanguageID = 1
)

var LanguageImages = map[LanguageID]string{
	Python: "python:3.12.12-slim",
}

func (l LanguageID) String() string {
	switch l {
	case Python:
		return "Python"
	default:
		return "Unknown"
	}
}

type WorkerConfigData struct {
	ContainerTimeout time.Duration
	TestTimeout      time.Duration
	MaximumRamMB     int
}

type Worker struct {
	client       *client.Client
	clientConfig *container.Config
	hostConfig   *container.HostConfig

	dataPath         string
	language         LanguageID
	maxRamMB         int
	testTimeout      time.Duration
	containerTimeout time.Duration
}

func NewWorker(config WorkerConfigData) (*Worker, error) {
	// inicia o client worker

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())

	return &Worker{
		client:           cli,
		containerTimeout: config.ContainerTimeout,
		testTimeout:      config.TestTimeout,
		maxRamMB:         config.MaximumRamMB,
	}, nil
}

// prepara o ambiente de execução, com os arquivos de .in e .out e meta.json

type DockerWorkspaceConfig struct {
	CachePath          string
	ExecutionDirectory string
	RunnerPath         string
	sourceCode         string
}

func (w *Worker) PrepareWorkspace(config DockerWorkspaceConfig) error {
	// garante que existe a pasta de execução
	if err := os.MkdirAll(config.ExecutionDirectory, 0755); err != nil {
		return err
	}

	// faz a pasta temporária
	tempPath, err := os.MkdirTemp(config.ExecutionDirectory, "job-*")
	if err != nil {
		return err
	}

	// pega caminho absoluto
	absPath, err := filepath.Abs(tempPath)
	if err != nil {
		os.RemoveAll(tempPath)
		return err
	}
	w.dataPath = absPath

	// copia os conteúdos da pasta de cache
	err = folderutils.CopyDir(config.CachePath, w.dataPath)
	if err != nil {
		w.Cleanup()
		return err
	}

	// copia o binário do runner
	if err := prepareRunnerBinary(config.RunnerPath, w.dataPath); err != nil {
		w.Cleanup()
		return err
	}

	return nil
}

func prepareRunnerBinary(sourcePath, destDir string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read runner binary: %w", err)
	}

	destPath := filepath.Join(destDir, "runner")

	// garante as permissões de execução
	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write executable runner: %w", err)
	}

	return nil
}

func (w *Worker) SetupPython(sourceCode string) error {
	w.language = Python

	fullPath := filepath.Join(w.dataPath, "source.py")

	err := os.WriteFile(fullPath, []byte(sourceCode), 0644)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo do código fonte: %w", err)
	}

	w.clientConfig = &container.Config{
		Image:      "python:3.12.12-slim",
		Cmd:        []string{"./runner", "python", "source.py"},
		WorkingDir: "/app",
	}

	w.hostConfig = &container.HostConfig{
		NetworkMode: "none",
		Resources: container.Resources{
			Memory: int64(w.maxRamMB * 1024 * 1024),
		},
		Binds: []string{
			fmt.Sprintf("%s:/app:rw", w.dataPath),
		},
	}

	return nil
}

func (w *Worker) SetupCustom(containerConfig *container.Config, hostConfig *container.HostConfig) {
	w.clientConfig = containerConfig
	w.hostConfig = hostConfig
}

func (w *Worker) Cleanup() {
	if w.dataPath != "" {
		os.RemoveAll(w.dataPath)
	}
}

func (w *Worker) Execute() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), w.containerTimeout)
	defer cancel()

	containerID, err := w.client.ContainerCreate(ctx, w.clientConfig, w.hostConfig, nil, nil, "")
	if err != nil {
		return "", err
	}
	defer w.client.ContainerRemove(context.Background(), containerID.ID, container.RemoveOptions{Force: true})

	err = w.client.ContainerStart(ctx, containerID.ID, container.StartOptions{})
	if err != nil {
		return "", err
	}

	statusCh, errCh := w.client.ContainerWait(ctx, containerID.ID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		// canal de erro -> erro
		if err != nil {
			return "", err
		}

	case <-statusCh:
		// terminou o container com sucesso
		break

	case <-ctx.Done():
		// timeout, contexto de timeout finalizado
		return "", fmt.Errorf("Timeout do Container excedido.")
	}

	resultPath := filepath.Join(w.dataPath, "result.json")
	content, err := os.ReadFile(resultPath)
	if err != nil {
		return "", fmt.Errorf("falha ao ler result.json (runner crashou?): %w", err)
	}

	return string(content), nil
}
