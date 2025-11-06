package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

var (
	codigo = "var1 = input()\nvar2 = input()\nvar3 = input()\nprint(var1, var2, var3)"
	input  = "teste\nteste\nteste\n"
)

func main() {
	// inicializa o contexto
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // <-- timeout de 5 segundos
	defer cancel()

	// inicializa o client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx)

	client_config := &container.Config{
		Image:        "python:3.12.12-slim",
		Cmd:          []string{"python", "-c", codigo},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		Tty:          false,
	}

	hostConfig := &container.HostConfig{
		NetworkMode: "none", // <- desliga rede
		Resources: container.Resources{
			Memory: 64 * 1024 * 1024, // 64MB de ram
		},
	}

	// cria o container
	containerID, err := cli.ContainerCreate(ctx, client_config, hostConfig, nil, nil, "")
	if err != nil {
		panic(err)
	}

	// limpa o container
	defer cli.ContainerRemove(context.Background(), containerID.ID, container.RemoveOptions{Force: true})

	// incializa o container
	err = cli.ContainerStart(ctx, containerID.ID, container.StartOptions{})
	if err != nil {
		panic(err)
	}

	// envia input para o container,
	go func() {
		// cria attach para enviar
		attach, _ := cli.ContainerAttach(ctx, containerID.ID, container.AttachOptions{Stream: true, Stdin: true})
		defer attach.CloseWrite()
		io.Copy(attach.Conn, strings.NewReader(input))
	}()

	// espera e cria os canais de erro e status
	statusCh, errCh := cli.ContainerWait(ctx, containerID.ID, container.WaitConditionNotRunning)

	select {

	case err := <-errCh:
		// canal de erro -> erro
		if err != nil {
			panic(err)
		}

	case <-statusCh:
		// terminou o container com sucesso
		break

	case <-ctx.Done():
		// timeout, contexto de timeout finalizado
		panic(ctx.Err())
	}

	// lê os logs
	logs, err := cli.ContainerLogs(ctx, containerID.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
	})
	if err != nil {
		panic(err)
	}
	defer logs.Close()

	var buf bytes.Buffer
	io.Copy(&buf, logs)
	fmt.Println(buf.String())
}
