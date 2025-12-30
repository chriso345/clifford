scripts_dir := "./scripts"

# List tasks available
default:
    @just --list --list-prefix " - "

# Run unit tests
test:
    go test ./... -count=1

# Coverage report
cover:
    {{ scripts_dir }}/testing/run_coverage.sh

# Lint (if golangci-lint is installed)
lint:
    golangci-lint run || true

# Install development tools
install-tools:
    {{ scripts_dir }}/tools/install_tools.sh

# Run the command line application
run *args:
    go run ./cmd/gspl {{ args }}

# Docs generation
docs:
    gomarkdoc ./... > docs/clifford.md
