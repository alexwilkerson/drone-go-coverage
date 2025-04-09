# Drone Go Coverage Plugin

A Drone CI plugin for enforcing code coverage thresholds in Go projects. This plugin runs tests with coverage, analyzes the results, and fails the build if coverage falls below the specified threshold.

## Features

- Global coverage threshold enforcement
- Per-package threshold customization
- HTML and JSON report generation
- Configurable test arguments and timeouts
- Support for subdirectory testing

## Usage

Add the following to your `.drone.yml` file:

```yaml
kind: pipeline
name: default

steps:
  - name: coverage
    image: alexwilkerson/drone-go-coverage:latest
    settings:
      threshold: 80  # Minimum coverage percentage required
      subdirectory: "path/to/project"  # Optional: test a specific directory
      verbose_output: true  # Optional: show detailed coverage information
```

## Configuration

### Basic Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `threshold` | Minimum coverage percentage required | (required if `per_package_threshold` not set) |
| `per_package_threshold` | JSON map of package patterns to thresholds | (required if `threshold` not set) |
| `subdirectory` | Directory to run tests in | `.` |
| `verbose_output` | Show detailed coverage information | `false` |
| `fail_build` | Fail the build if threshold is not met | `true` |

### Advanced Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `include_packages` | Comma-separated list of packages to include | `./...` |
| `exclude_packages` | Comma-separated list of packages to exclude | |
| `cover_mode` | Coverage mode (atomic, count, set) | `atomic` |
| `test_timeout` | Test execution timeout | |
| `test_args` | Additional arguments to pass to `go test` | |
| `output_dir` | Directory for coverage reports | |
| `cover_profile` | Name of the coverage profile | `drone-go-coverage.out` |
| `html_report` | Generate HTML coverage report | `false` |
| `json_report` | Generate JSON coverage report | `false` |

## Examples

### Basic Coverage Testing

```yaml
steps:
  - name: coverage
    image: alexwilkerson/drone-go-coverage:latest
    settings:
      threshold: 80
```

### Advanced Configuration

`per_package_threshold` uses regex pattern matching and should match the full package name.

```yaml
steps:
  - name: coverage
    image: alexwilkerson/drone-go-coverage:latest
    settings:
      threshold: 75
      per_package_threshold: '{"github.com/org/repo/internal": 65, "github.com/org/repo/models": 90}'
      include_packages: "github.com/org/repo/..."
      exclude_packages: "github.com/org/repo/vendor/..."
      test_timeout: "5m"
      test_args: "-race -count=1"
      html_report: true
      json_report: true
      output_dir: "coverage"
      verbose_output: true
```

## Development

### Prerequisites

- Go 1.24 or later
- Docker for building and testing container images
- [Task](https://taskfile.dev/) for running development commands

### Building Locally

```bash
# Build the binary for local testing
task build

# Run Docker test
task test-docker
```

### Docker Image

```bash
# Build the Docker image
task docker-build

# Push the Docker image
task docker-push

# Create release tags for git and Docker
task create-release

# Publish the Docker image
task publish
```

### Releasing

To create a new release:

```bash
# Create version 0.2.0
task create-release -- 0.2.0
task publish
```

This will:
1. Create git tag v0.2.0
2. Push the tag to the remote repository
3. Build and publish Docker image alexwilkerson/drone-go-coverage:0.2.0
4. Also tag the image as alexwilkerson/drone-go-coverage:latest

## License

[MIT License](LICENSE)