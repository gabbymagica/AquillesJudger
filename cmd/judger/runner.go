package main

import (
	"context"
	"encoding/json"
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
	languageCommand := os.Args[1]
	otherArgs := os.Args[2:]

	file, err := os.ReadFile("inputs.json")
	if err != nil {
		panic(err)
	}

	//fmt.Println(string(file))

	var inputs Input
	err = json.Unmarshal(file, &inputs)
	if err != nil {
		panic(err)
	}

	//fmt.Println(inputs)

	outputs := run(languageCommand, otherArgs, inputs.Inputs)
	fmt.Println(outputs)
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
