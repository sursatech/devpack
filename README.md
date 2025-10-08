# Railpack

A tool for extracting development configuration from codebases.

## Build

```bash
go build -o bin/railpack ./cmd/cli
```

## Usage

Extract development configuration from any codebase:

```bash
./bin/railpack dev-config <DIRECTORY>
```

### Example Output

```json
{
  "detectedLanguage": "node",
  "aptPackages": "ca-certificates, fonts-liberation",
  "installCommand": "npm install",
  "startCommandHost": "npm run dev -- -H 0.0.0.0",
  "requiredPort": "3000",
  "variables": {
    "CI": "false",
    "HOSTNAME": "0.0.0.0",
    "NODE_ENV": "development",
    "NPM_CONFIG_FUND": "false",
    "NPM_CONFIG_PRODUCTION": "false",
    "NPM_CONFIG_UPDATE_NOTIFIER": "false"
  }
}
```

### Options

```bash
# Pretty output format
./bin/railpack dev-config <DIRECTORY> --format=pretty

# Save to file
./bin/railpack dev-config <DIRECTORY> --out config.json
```

### Examples

```bash
./bin/railpack dev-config examples/node-next
./bin/railpack dev-config examples/python-django
./bin/railpack dev-config examples/go-mod
```

## Testing

Run tests:
```bash
go test ./...
```

## CI/CD

The project uses GitHub Actions for automated testing. Push to `dev-pack-v2` branch to trigger the CI/CD pipeline.