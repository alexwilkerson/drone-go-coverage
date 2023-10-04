package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("[drone-go-coverage]")

	thresholdStr := os.Getenv("PLUGIN_COVERAGE_THRESHOLD")

	if thresholdStr == "" {
		fmt.Println("No coverage threshold set")
		os.Exit(1)
	}

	threshold, err := strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		fmt.Printf("Failed to parse coverage threshold: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Running tests...")

	cmd := exec.Command("go", "test", "-coverprofile=drone-go-coverage.out", "./...")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Tests failed")
		fmt.Println(string(output))
		os.Exit(1)
	}

	cmd = exec.Command("go", "tool", "cover", "-func=drone-go-coverage.out")
	output, err = cmd.Output()
	if err != nil {
		fmt.Printf("Failed to extract coverage: %v\n", err)
		fmt.Println(string(output))
		os.Exit(1)
	}

	err = os.Remove("drone-go-coverage.out")
	if err != nil {
		fmt.Printf("Failed to remove coverage.out: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))

	lines := strings.Split(string(output), "\n")
	var coverageLine string
	for _, line := range lines {
		if strings.Contains(line, "total:") {
			coverageLine = line
			break
		}
	}

	if coverageLine == "" {
		fmt.Println("Failed to find coverage line. Perhaps there are no tests?")
		os.Exit(1)
	}

	parts := strings.Fields(coverageLine)
	coverageStr := strings.TrimSuffix(parts[len(parts)-1], "%")
	coverage, err := strconv.ParseFloat(coverageStr, 64)
	if err != nil {
		fmt.Printf("Failed to parse coverage: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Total coverage: %.2f%%\n", coverage)

	if coverage < threshold {
		fmt.Printf("Coverage %.2f%% is below the threshold of %.2f%%\n", coverage, threshold)
		os.Exit(1)
	}

	fmt.Printf("Coverage is above the threshold of %.2f%%\n", threshold)
}
