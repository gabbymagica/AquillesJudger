package worker

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// enum linguagens
const (
	Python = 1
)

var runnerLanguage = map[int][]string{
	Python: {"python", "-c", ""},
}

type Worker struct {
	client       *client.Client
	clientConfig *container.Config
	hostConfig   *container.HostConfig
	language     int
}

func NewWorker() (*Worker, error) {
	// inicia o client worker

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())

	return &Worker{client: cli}, nil
}

func (w *Worker) injectCode(codigo string) {
	switch w.language {
	case Python:
		w.clientConfig.Cmd[2] = codigo
	}
}

func (w *Worker) SetupPython(ramMB int64) {
	w.language = Python

	w.clientConfig = &container.Config{
		Image:        "python:3.12.12-slim",
		Cmd:          runnerLanguage[w.language],
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		Tty:          false,
	}

	w.hostConfig = &container.HostConfig{
		NetworkMode: "none", // <- desliga rede
		Resources: container.Resources{
			Memory: ramMB * 1024 * 1024,
		},
	}

}

func (w *Worker) SetupCustom(containerConfig *container.Config, hostConfig *container.HostConfig) {
	w.clientConfig = containerConfig
	w.hostConfig = hostConfig
}

func (w *Worker) Execute(codigo string, input string, timeout time.Duration) (stdOut string, stdErr string, error error) {
	if input != "" {
		if input[len(input)-1] != '\n' {
			input += "\n"
		}
	}

	w.injectCode(codigo)

	// timeout context
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Second)
	defer cancel()

	// cria o container
	containerID, err := w.client.ContainerCreate(ctx, w.clientConfig, w.hostConfig, nil, nil, "")
	if err != nil {
		return "", "", err
	}

	// garante o extermínio do container
	defer w.client.ContainerRemove(context.Background(), containerID.ID, container.RemoveOptions{Force: true})

	// incializa o container
	err = w.client.ContainerStart(ctx, containerID.ID, container.StartOptions{})
	if err != nil {
		return "", "", err
	}

	// goroutine pra enviar input pro container
	go func() {
		// inicia o link de escrita
		attach, _ := w.client.ContainerAttach(ctx, containerID.ID, container.AttachOptions{Stream: true, Stdin: true})

		// garante a limpeza do attach
		defer attach.CloseWrite()

		// bloqueante, espera pedido de stdin para enviar input
		io.Copy(attach.Conn, strings.NewReader(input))
		//fmt.Println("Input enviado")
	}()

	// inicia o link de leitura
	attachReader, err := w.client.ContainerAttach(ctx, containerID.ID, container.AttachOptions{Stream: true, Stdout: true, Stderr: true})
	if err != nil {
		return "", "", err
	}

	// inicia os canais para status
	statusCh, errCh := w.client.ContainerWait(ctx, containerID.ID, container.WaitConditionNotRunning)

	select {

	case err := <-errCh:
		// canal de erro -> erro
		if err != nil {
			return "", "", err
		}

	case <-statusCh:
		// terminou o container com sucesso
		break

	case <-ctx.Done():
		// timeout, contexto de timeout finalizado
		return "", "", ctx.Err()
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, attachReader.Reader)
	if err != nil {
		return "", "error while reading output", err
	}

	return stdoutBuf.String(), stderrBuf.String(), nil
}
