package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

type Input struct {
	Inputs []string `json:"usecases"`
}

func main() {
	fmt.Println(os.Args)
}

func run(cmd string, otherArgs, inputs []string) []string {
	var outputs []string
	for i := range inputs {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		cmd := exec.CommandContext(timeoutCtx, cmd, otherArgs...)

		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			defer stdin.Close()
			io.WriteString(stdin, inputs[i])
		}()

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatal(err)
		}

		outputs = append(outputs, string(out))
		cancel()
	}

	return outputs
}
