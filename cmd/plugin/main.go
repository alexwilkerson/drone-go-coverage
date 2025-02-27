package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Config holds the plugin configuration
type Config struct {
	Threshold           float64            `json:"threshold"`
	PerPackageThreshold map[string]float64 `json:"per_package_threshold"`
	Subdirectory        string             `json:"subdirectory"`
	OutputFormat        string             `json:"output_format"`
	IncludePackages     []string           `json:"include_packages"`
	ExcludePackages     []string           `json:"exclude_packages"`
	TestTimeout         string             `json:"test_timeout"`
	TestArgs            []string           `json:"test_args"`
	FailBuild           bool               `json:"fail_build"`
	CoverMode           string             `json:"cover_mode"`
	OutputDir           string             `json:"output_dir"`
	CoverProfile        string             `json:"cover_profile"`
	HTMLReport          bool               `json:"html_report"`
	JSONReport          bool               `json:"json_report"`
	VerboseOutput       bool               `json:"verbose_output"`
}

const (
	statusPass = "pass"
	statusFail = "fail"
)

// CoverageResult represents a single file/package coverage result
type CoverageResult struct {
	Package  string  `json:"package"`
	Coverage float64 `json:"coverage"`
	Status   string  `json:"status"` // pass, fail
}

// FullReport represents the complete coverage report
type FullReport struct {
	Timestamp      string           `json:"timestamp"`
	TotalCoverage  float64          `json:"total_coverage"`
	Threshold      float64          `json:"threshold"`
	Status         string           `json:"status"` // pass, fail
	PackageResults []CoverageResult `json:"package_results"`
}

func main() {
	ctx := context.Background()

	fmt.Println("[drone-go-coverage] Starting coverage analysis")

	config, err := loadConfig()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	if config.VerboseOutput {
		printConfig(config)
	}

	if config.Subdirectory != "" {
		fmt.Printf("Changing to subdirectory: %s\n", config.Subdirectory)
		if err := os.Chdir(config.Subdirectory); err != nil {
			fmt.Printf("failed to change directory: %v\n", err)
			os.Exit(1)
		}
	}

	if config.OutputDir != "" {
		if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
			fmt.Printf("failed to create output directory: %v\n", err)
			os.Exit(1)
		}
	}

	coverProfile := "drone-go-coverage.out"
	if config.CoverProfile != "" {
		coverProfile = config.CoverProfile
	}

	if err := runTests(ctx, config, coverProfile); err != nil {
		fmt.Printf("failed to run tests: %v\n", err)
		os.Exit(1)
	}

	results, totalCoverage, err := extractCoverage(ctx, coverProfile)
	if err != nil {
		fmt.Printf("failed to extract coverage: %v\n", err)
		os.Exit(1)
	}

	report := generateReport(config, results, totalCoverage)

	outputResults(ctx, config, report, coverProfile)

	if err := os.Remove(coverProfile); err != nil {
		fmt.Printf("failed to remove cover profile: %v\n", err)
	} else if config.VerboseOutput {
		fmt.Printf("[drone-go-coverage] Removed coverage profile: %s\n", coverProfile)
	}

	if report.Status == statusFail && config.FailBuild {
		if report.TotalCoverage < config.Threshold {
			fmt.Printf("[drone-go-coverage] Coverage %.2f%% is below the threshold of %.2f%%\n", report.TotalCoverage, config.Threshold)
		} else {
			fmt.Printf("[drone-go-coverage] Some packages have coverage below their threshold requirements\n")
		}
		os.Exit(1)
	}

	fmt.Printf("[drone-go-coverage] Overall coverage: %.2f%% (threshold: %.2f%%)\n", report.TotalCoverage, config.Threshold)
	fmt.Println("[drone-go-coverage] Analysis complete")
}

func loadConfig() (Config, error) {
	config := Config{
		FailBuild:     true,
		CoverMode:     "atomic",
		OutputFormat:  "text",
		VerboseOutput: false,
	}

	thresholdStr := os.Getenv("PLUGIN_THRESHOLD")
	if thresholdStr != "" {
		threshold, err := strconv.ParseFloat(thresholdStr, 64)
		if err != nil {
			return config, fmt.Errorf("failed to parse threshold: %v", err)
		}
		config.Threshold = threshold
	}

	perPackageThreshold := os.Getenv("PLUGIN_PER_PACKAGE_THRESHOLD")
	if perPackageThreshold != "" {
		config.PerPackageThreshold = make(map[string]float64)

		if err := json.Unmarshal([]byte(perPackageThreshold), &config.PerPackageThreshold); err != nil {
			return config, fmt.Errorf("failed to parse per-package threshold: %v", err)
		}
	}

	config.Subdirectory = os.Getenv("PLUGIN_SUBDIRECTORY")
	config.OutputFormat = os.Getenv("PLUGIN_OUTPUT_FORMAT")
	config.TestTimeout = os.Getenv("PLUGIN_TEST_TIMEOUT")
	config.OutputDir = os.Getenv("PLUGIN_OUTPUT_DIR")
	config.CoverProfile = os.Getenv("PLUGIN_COVER_PROFILE")
	config.CoverMode = os.Getenv("PLUGIN_COVER_MODE")

	config.HTMLReport = getEnvBool("PLUGIN_HTML_REPORT", false)
	config.JSONReport = getEnvBool("PLUGIN_JSON_REPORT", false)
	config.FailBuild = getEnvBool("PLUGIN_FAIL_BUILD", true)
	config.VerboseOutput = getEnvBool("PLUGIN_VERBOSE_OUTPUT", false)

	if packages := os.Getenv("PLUGIN_INCLUDE_PACKAGES"); packages != "" {
		config.IncludePackages = strings.Split(packages, ",")
	}

	if packages := os.Getenv("PLUGIN_EXCLUDE_PACKAGES"); packages != "" {
		config.ExcludePackages = strings.Split(packages, ",")
	}

	if args := os.Getenv("PLUGIN_TEST_ARGS"); args != "" {
		config.TestArgs = strings.Fields(args)
	}

	if config.Threshold == 0 && len(config.PerPackageThreshold) == 0 {
		return config, fmt.Errorf("no threshold or per-package threshold set")
	}

	return config, nil
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		parsedValue, err := strconv.ParseBool(value)
		if err != nil {
			fmt.Printf("failed to parse %s: %v\nusing default value of %v\n", key, err, defaultValue)
			return defaultValue
		}
		return parsedValue
	}
	return defaultValue
}

func printConfig(config Config) {
	fmt.Println("drone-go-coverage Plugin Configuration:")
	fmt.Printf("  - Global Threshold: %.2f%%\n", config.Threshold)
	if len(config.PerPackageThreshold) > 0 {
		fmt.Println("  - Per-Package Thresholds:")
		for pkg, threshold := range config.PerPackageThreshold {
			fmt.Printf("    - %s: %.2f%%\n", pkg, threshold)
		}
	}
	if config.Subdirectory != "" {
		fmt.Printf("  - Subdirectory: %s\n", config.Subdirectory)
	}
	fmt.Printf("  - Output Format: %s\n", config.OutputFormat)
	if config.HTMLReport {
		fmt.Println("  - HTML Report: enabled")
	}
	if config.JSONReport {
		fmt.Println("  - JSON Report: enabled")
	}
	fmt.Printf("  - Fail Build: %v\n", config.FailBuild)
	if config.TestTimeout != "" {
		fmt.Printf("  - Test Timeout: %s\n", config.TestTimeout)
	}
	if len(config.IncludePackages) > 0 {
		fmt.Printf("  - Include Packages: %s\n", strings.Join(config.IncludePackages, ", "))
	}
	if len(config.ExcludePackages) > 0 {
		fmt.Printf("  - Exclude Packages: %s\n", strings.Join(config.ExcludePackages, ", "))
	}
	if config.OutputDir != "" {
		fmt.Printf("  - Output Directory: %s\n", config.OutputDir)
	}
	if config.CoverProfile != "" {
		fmt.Printf("  - Cover Profile: %s\n", config.CoverProfile)
	}
	fmt.Printf("  - Cover Mode: %s\n", config.CoverMode)
}

func runTests(ctx context.Context, config Config, coverProfile string) error {
	fmt.Println("[drone-go-coverage] Running tests with coverage...")

	args := []string{"test", "-coverprofile=" + coverProfile, "-covermode=" + config.CoverMode}

	if config.TestTimeout != "" {
		args = append(args, fmt.Sprintf("-timeout=%s", config.TestTimeout))
	}

	if len(config.TestArgs) > 0 {
		args = append(args, config.TestArgs...)
	}

	if len(config.IncludePackages) > 0 {
		args = append(args, config.IncludePackages...)
	} else {
		args = append(args, "./...")
	}

	if config.VerboseOutput {
		fmt.Printf("[drone-go-coverage] Running command: go %s\n", strings.Join(args, " "))
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	output, err := cmd.CombinedOutput()

	// always print output, regardless of error
	fmt.Println(string(output))

	if err != nil {
		return fmt.Errorf("failed to run tests: %v", err)
	}

	return nil
}

func extractCoverage(ctx context.Context, coverProfile string) ([]CoverageResult, float64, error) {
	cmd := exec.CommandContext(ctx, "go", "tool", "cover", "-func="+coverProfile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to extract coverage: %v", err)
	}

	lines := strings.Split(string(output), "\n")

	packageCoverageSum := make(map[string]float64)
	packageFuncCount := make(map[string]int)

	var (
		totalCoverage float64
		coverageLine  string
	)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		if strings.Contains(line, "total:") {
			coverageLine = line
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 3 {
			pkgPath := parts[0]

			if idx := strings.LastIndex(pkgPath, ":"); idx != -1 {
				pkgPath = pkgPath[:idx]
			}

			pkgName := pkgPath
			if idx := strings.LastIndex(pkgPath, "/"); idx != -1 {
				pkgName = filepath.Dir(pkgPath)
			}

			coverageStr := strings.TrimSuffix(parts[len(parts)-1], "%")
			coverage, err := strconv.ParseFloat(coverageStr, 64)
			if err != nil {
				continue // skip malformed lines
			}

			packageCoverageSum[pkgName] += coverage
			packageFuncCount[pkgName]++
		}
	}

	results := make([]CoverageResult, 0, len(packageCoverageSum))
	for pkg, sum := range packageCoverageSum {
		avgCoverage := sum / float64(packageFuncCount[pkg])
		results = append(results, CoverageResult{
			Package:  pkg,
			Coverage: avgCoverage,
			Status:   statusPass,
		})
	}

	if coverageLine != "" {
		parts := strings.Fields(coverageLine)
		coverageStr := strings.TrimSuffix(parts[len(parts)-1], "%")
		totalCoverage, err = strconv.ParseFloat(coverageStr, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse total coverage: %w", err)
		}
	} else {
		return nil, 0, fmt.Errorf("failed to find total coverage line")
	}

	return results, totalCoverage, nil
}

func generateReport(config Config, results []CoverageResult, totalCoverage float64) FullReport {
	report := FullReport{
		Timestamp:      time.Now().Format(time.RFC3339),
		TotalCoverage:  totalCoverage,
		Threshold:      config.Threshold,
		Status:         statusPass,
		PackageResults: results,
	}

	if totalCoverage < config.Threshold {
		report.Status = statusFail
	}

	for i, result := range report.PackageResults {
		threshold := config.Threshold
		for pattern, pkgThreshold := range config.PerPackageThreshold {
			match, err := regexp.MatchString(pattern, result.Package)
			if err != nil {
				fmt.Printf("failed to match package pattern: %v\n", err)
				continue
			}

			if match {
				threshold = pkgThreshold
				break
			}
		}

		if result.Coverage < threshold {
			report.PackageResults[i].Status = statusFail
			if config.FailBuild {
				report.Status = statusFail
			}
		}
	}

	return report
}

func outputResults(ctx context.Context, config Config, report FullReport, coverProfile string) {
	fmt.Println("[drone-go-coverage] Coverage Results:")
	fmt.Println("--------------------------------------------------")
	fmt.Printf("Total Coverage: %.2f%% (Threshold: %.2f%%)\n", report.TotalCoverage, report.Threshold)
	fmt.Printf("Status: %s\n", strings.ToUpper(report.Status))
	fmt.Println("--------------------------------------------------")

	if config.VerboseOutput {
		fmt.Println("[drone-go-coverage] Package Coverage Details:")
		fmt.Println("Package                                   Coverage  Status")
		fmt.Println("--------------------------------------------------")
		for _, result := range report.PackageResults {
			fmt.Printf("%-40s %.2f%%   %s\n", result.Package, result.Coverage, strings.ToUpper(result.Status))
		}
		fmt.Println("--------------------------------------------------")
	}

	if config.JSONReport {
		outputJSON(config, report)
	}

	if config.HTMLReport {
		outputHTML(ctx, config, coverProfile)
	}
}

func outputJSON(config Config, report FullReport) {
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Printf("failed to generate JSON report: %v\n", err)
		return
	}

	outputPath := "coverage-report.json"
	if config.OutputDir != "" {
		outputPath = filepath.Join(config.OutputDir, outputPath)
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		fmt.Printf("failed to write JSON report: %v\n", err)
		return
	}

	fmt.Printf("JSON report written to: %s\n", outputPath)
}

func outputHTML(ctx context.Context, config Config, coverProfile string) {
	outputPath := "coverage-report.html"
	if config.OutputDir != "" {
		outputPath = filepath.Join(config.OutputDir, outputPath)
	}

	cmd := exec.CommandContext(ctx, "go", "tool", "cover", "-html="+coverProfile, "-o="+outputPath)
	if err := cmd.Run(); err != nil {
		fmt.Printf("failed to generate HTML report: %v\n", err)
		return
	}

	fmt.Printf("HTML report written to: %s\n", outputPath)
}
