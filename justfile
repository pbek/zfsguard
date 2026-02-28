# Use `just <recipe>` to run a recipe
# https://just.systems/man/en/

import ".shared/common.just"

version := `cat VERSION`

# By default, run the `--list` command
default:
    @just --list

# Download dependencies
[group('build')]
dep:
    go mod download

# Update dependencies
[group('build')]
update:
    go get -u
    go mod tidy

# Build both binaries
[group('build')]
build:
    go build -ldflags "-X github.com/pbek/zfsguard/internal/version.Version={{ version }}" -v ./cmd/zfsguard
    go build -ldflags "-X github.com/pbek/zfsguard/internal/version.Version={{ version }}" -v ./cmd/zfsguard-monitor

# Install both binaries
[group('build')]
install:
    go install -ldflags "-X github.com/pbek/zfsguard/internal/version.Version={{ version }}" ./cmd/zfsguard
    go install -ldflags "-X github.com/pbek/zfsguard/internal/version.Version={{ version }}" ./cmd/zfsguard-monitor

# Run tests
[group('test')]
test:
    go test -v -race ./...

# Run go vet
[group('test')]
vet:
    go vet ./...

# Build using nix
[group('nix')]
nix-build:
    nix build

# Build using nix (force rebuild)
[group('nix')]
nix-build-force:
    nix build --rebuild
