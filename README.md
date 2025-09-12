# Railpack

[![CI](https://github.com/railwayapp/railpack/actions/workflows/ci.yml/badge.svg)](https://github.com/railwayapp/railpack/actions/workflows/ci.yml)
[![Run Tests](https://github.com/railwayapp/railpack/actions/workflows/run_tests.yml/badge.svg)](https://github.com/railwayapp/railpack/actions/workflows/run_tests.yml)

Railpack is a tool for building images from source code with minimal
configuration. It is the successor to [Nixpacks](https://nixpacks.com) and
incorporates many of the learnings from running Nixpacks in production at
[Railway](https://railway.com) for several years.

## Documentation

Railpack is an early work in progress and constantly evolving. Documentation for
both operators and users is available at [railpack.com](https://railpack.com).

## Contributing

Railpack is open source and open to contributions. Keep in mind that the project
is early and core architectural changes are still being made.

See the [CONTRIBUTING.md](CONTRIBUTING.md) file for more information.

## Run locally

### Prerequisites

- Go (1.22+ recommended)
- Optional (for building images): a running BuildKit daemon and `BUILDKIT_HOST` set
  - Quick start BuildKit with Docker:
    - `docker run --rm --privileged -d --name buildkit moby/buildkit`
    - `export BUILDKIT_HOST='docker-container://buildkit'`

### Build the CLI

```bash
go build -o bin/railpack ./cmd/cli
./bin/railpack --help
```

### Analyze an app and get run commands

Pretty output (with metadata):

```bash
./bin/railpack info <DIRECTORY> --format=pretty
```

JSON output (machine-readable):

```bash
./bin/railpack info <DIRECTORY> --format=json > plan-info.json
```

Development mode (local run):

```bash
./bin/railpack info <DIRECTORY> --dev --format=pretty
```

Key fields in output:

- `deploy.startCommand` — command to run the app
- `deploy.variables` — environment variables to set
- `steps[]` — build/install graph

### Generate a standalone plan file

```bash
./bin/railpack plan <DIRECTORY> --out .railpack/plan.json
```

This includes `$schema` so editors can validate the JSON.

### Build an image with BuildKit

Ensure `BUILDKIT_HOST` is set (see prerequisites), then:

```bash
./bin/railpack build <DIRECTORY> \
  --name my-app:latest \
  --platform linux/amd64 \
  --progress tty
```

Export final filesystem instead of an image:

```bash
./bin/railpack build <DIRECTORY> --output out-fs
```

Show the build plan before building:

```bash
./bin/railpack build <DIRECTORY> --show-plan
```

### Useful flags (all commands)

- `--dev` — generate development config (local-run commands/env)
- `--env KEY=VALUE` — inject config/test envs (repeatable)
- `--build-cmd "..."` — override build command
- `--start-cmd "..."` — override start command
- `--config-file path/to/railpack.json` — custom config path
- `--error-missing-start` — fail if no start command is found

### Dev vs Prod behavior (high-level)

- Node: `--dev` prefers framework/package-manager dev scripts; SPA apps skip static server in dev
- Python: `--dev` prefers live servers (Django runserver, uvicorn --reload, flask run)
- Deno: `--dev` uses `deno task dev` when present
- Go: `--dev` uses `go run` heuristics matching your layout
- Java: `--dev` uses `gradle run` or `mvn spring-boot:run` when applicable
- PHP: `--dev` uses `php artisan serve` (Laravel) or `php -S` for vanilla

### Work with examples

```bash
# Pretty info (prod)
./bin/railpack info examples/node-next --format=pretty

# Dev info
./bin/railpack info examples/node-next --dev --format=pretty

# Plan (json)
./bin/railpack plan examples/go-mod --out .railpack/plan.json
```

### Troubleshooting

- "BUILDKIT_HOST environment variable is not set": start BuildKit and export `BUILDKIT_HOST` as shown above.
- Use `--verbose` (global flag) to enable more logs: `./bin/railpack --verbose info <DIR>`
