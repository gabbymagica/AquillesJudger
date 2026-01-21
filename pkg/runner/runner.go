package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

func main() {
	userCmd, timeout := parseArgs()

	inputs, err := findTestInputs(".")
	if err != nil {
		writeErrorAndExit(err)
	}

	var results []TestCaseResult

	for _, inputDesc := range inputs {
		res := runTestCase(inputDesc, userCmd, timeout)
		results = append(results, res)
	}

	report := ExecutionReport{Results: results}
	if err := saveReport(report); err != nil {
		fmt.Fprintf(os.Stderr, "Falha ao salvar relatório: %v\n", err)
		os.Exit(1)
	}
}

type TestPair struct {
	ID         string
	InputPath  string
	OutputPath string
}

func parseArgs() ([]string, time.Duration) {
	var cmd []string
	var timeout time.Duration = 2 * time.Second

	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--testTimeout=") {
			valStr := strings.TrimPrefix(arg, "--testTimeout=")
			if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
				timeout = time.Duration(val)
			}
		} else {
			cmd = append(cmd, arg)
		}
	}

	if len(cmd) == 0 {
		fmt.Println("Nenhum comando fornecido")
		os.Exit(1)
	}

	return cmd, timeout
}

func findTestInputs(dir string) ([]TestPair, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var tests []TestPair
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".in") {
			id := strings.TrimSuffix(e.Name(), ".in")

			tests = append(tests, TestPair{
				ID:         id,
				InputPath:  filepath.Join(dir, e.Name()),
				OutputPath: filepath.Join(dir, id+".out"),
			})
		}
	}
	return tests, nil
}

func runTestCase(test TestPair, cmdArgs []string, timeout time.Duration) TestCaseResult {
	result := TestCaseResult{ID: test.ID}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)

	inputFile, err := os.Open(test.InputPath)
	if err != nil {
		result.Status = "IER"
		result.Message = fmt.Sprintf("Failed to open input: %v", err)
		return result
	}
	defer inputFile.Close()
	cmd.Stdin = inputFile

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	duration := time.Since(start)
	result.TimeMS = duration.Milliseconds()

	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "TLE"
		return result
	}

	if err != nil {
		result.Status = "RTE"
		msg := stderr.String()
		if len(msg) > 1000 {
			msg = msg[:1000] + "... (truncated)"
		}
		result.Message = msg
		return result
	}

	expectedBytes, err := os.ReadFile(test.OutputPath)
	if err != nil {
		result.Status = "IER"
		result.Message = "Expected output file not found"
		return result
	}

	userOutput := normalizeString(stdout.String())
	expectedOutput := normalizeString(string(expectedBytes))

	if userOutput == expectedOutput {
		result.Status = "AC"
	} else {
		result.Status = "WA"
		result.Message = fmt.Sprintf("Expected len %d, got %d", len(expectedOutput), len(userOutput))
	}

	return result
}

func normalizeString(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

func saveReport(report ExecutionReport) error {
	file, err := os.Create("result.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeErrorAndExit(err error) {
	report := ExecutionReport{
		Results: []TestCaseResult{
			{ID: "0", Status: "IER", Message: err.Error()},
		},
	}
	saveReport(report)
	os.Exit(1)
}
