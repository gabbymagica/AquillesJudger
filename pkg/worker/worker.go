package worker

import (
	folderutils "IFJudger/pkg/folder_utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type LanguageID int

const (
	Python LanguageID = 1
)

type ExecutionReport struct {
	Results []TestCaseResult `json:"results"`
}

type TestCaseResult struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	TimeMS  int64  `json:"time_ms"`
	Message string `json:"message,omitempty"`
}

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

type DockerWorkspaceConfig struct {
	CachePath          string
	ExecutionDirectory string
	RunnerPath         string
}

func (w *Worker) PrepareWorkspace(config DockerWorkspaceConfig) error {
	if err := os.MkdirAll(config.ExecutionDirectory, 0755); err != nil {
		return err
	}

	tempPath, err := os.MkdirTemp(config.ExecutionDirectory, "job-*")
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(tempPath)
	if err != nil {
		os.RemoveAll(tempPath)
		return err
	}
	w.dataPath = absPath

	err = folderutils.CopyDir(config.CachePath, w.dataPath)
	if err != nil {
		w.Cleanup()
		return err
	}

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
		Cmd:        []string{"./runner", "python", "source.py", fmt.Sprintf("--testTimeout=%d", w.testTimeout)},
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

func (w *Worker) Execute() (ExecutionReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), w.containerTimeout)
	defer cancel()

	containerID, err := w.client.ContainerCreate(ctx, w.clientConfig, w.hostConfig, nil, nil, "")
	if err != nil {
		return ExecutionReport{}, err
	}
	defer w.client.ContainerRemove(context.Background(), containerID.ID, container.RemoveOptions{Force: true})

	err = w.client.ContainerStart(ctx, containerID.ID, container.StartOptions{})
	if err != nil {
		return ExecutionReport{}, err
	}

	statusCh, errCh := w.client.ContainerWait(ctx, containerID.ID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return ExecutionReport{}, err
		}

	case <-statusCh:
		break

	case <-ctx.Done():
		return ExecutionReport{}, fmt.Errorf("Timeout do Container excedido.")
	}

	resultPath := filepath.Join(w.dataPath, "result.json")
	content, err := os.ReadFile(resultPath)
	if err != nil {
		out, errLogs := w.client.ContainerLogs(context.Background(), containerID.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		})

		if errLogs == nil {
			var stdoutBuf, stderrBuf bytes.Buffer
			stdcopy.StdCopy(&stdoutBuf, &stderrBuf, out)

			return ExecutionReport{}, fmt.Errorf(
				"result.json not found. Container Logs:\nSTDOUT: %s\nSTDERR: %s\nOriginal Error: %w",
				stdoutBuf.String(),
				stderrBuf.String(),
				err,
			)
		}

		return ExecutionReport{}, fmt.Errorf("falha ao ler result.json (runner crashou?): %w", err)
	}

	var executionReport ExecutionReport
	if err := json.Unmarshal(content, &executionReport); err != nil {
		return ExecutionReport{}, err
	}

	return executionReport, nil
}
