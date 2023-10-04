### Drone configuration example

```yaml
kind: pipeline
name: default

steps:
  - name: coverage-retention
    image: alexwilkerson/drone-go-coverage:latest
    settings:
      coverage_threshold: 49
