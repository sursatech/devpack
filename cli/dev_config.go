package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/railwayapp/railpack/core"
	a "github.com/railwayapp/railpack/core/app"
	"github.com/railwayapp/railpack/core/plan"
	"github.com/railwayapp/railpack/internal/utils"
	"github.com/urfave/cli/v3"
)

type DevConfigOutput struct {
	DetectedLanguage string            `json:"detectedLanguage"`
	AptPackages      string            `json:"aptPackages"`
	InstallCommand   string            `json:"installCommand"`
	StartCommandHost string            `json:"startCommandHost"`
	RequiredPort     string            `json:"requiredPort"`
	Variables        map[string]string `json:"variables"`
}

var DevConfigCommand = &cli.Command{
	Name:                  "dev-config",
	Aliases:               []string{"dev"},
	Usage:                 "extract simplified development configuration for a codebase",
	ArgsUsage:             "DIRECTORY",
	EnableShellCompletion: true,
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "format",
			Usage: "output format. one of: pretty, json",
			Value: "json",
		},
		&cli.StringFlag{
			Name:  "out",
			Usage: "output file name",
		},
	}, commonPlanFlags()...),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		buildResult, _, _, err := GenerateDevBuildResultForCommand(cmd)
		if err != nil {
			return cli.Exit(err, 1)
		}

		if !buildResult.Success {
			log.Error("Failed to generate build plan")
			for _, logMsg := range buildResult.Logs {
				log.Error(logMsg.Msg)
			}
			return cli.Exit("Failed to generate build plan", 1)
		}

		config := extractDevConfig(buildResult)

		var output string
		format := cmd.String("format")

		if format == "pretty" {
			output = formatDevConfigPretty(config)
		} else {
			jsonBytes, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return cli.Exit(fmt.Errorf("error marshaling config: %w", err), 1)
			}
			output = string(jsonBytes)
		}

		// Output to file or stdout
		outFile := cmd.String("out")
		if outFile != "" {
			if err := os.WriteFile(outFile, []byte(output), 0644); err != nil {
				return cli.Exit(fmt.Errorf("error writing to file: %w", err), 1)
			}
			log.Infof("Configuration written to %s", outFile)
		} else {
			fmt.Println(output)
		}

		return nil
	},
}

// GenerateDevBuildResultForCommand is like GenerateBuildResultForCommand but forces dev mode
func GenerateDevBuildResultForCommand(cmd *cli.Command) (*core.BuildResult, *a.App, *a.Environment, error) {
	directory := cmd.Args().First()

	if directory == "" {
		return nil, nil, nil, cli.Exit("directory argument is required", 1)
	}

	app, err := a.NewApp(directory)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating app: %w", err)
	}

	log.Debugf("Building %s", app.Source)

	envsArgs := cmd.StringSlice("env")

	env, err := a.FromEnvs(envsArgs)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating env: %w", err)
	}

	if cmd.Bool("verbose") && env.GetVariable("MISE_VERBOSE") == "" {
		env.SetVariable("MISE_VERBOSE", "1")
	}

	previousVersions := utils.ParsePackageWithVersion(cmd.StringSlice("previous"))

	generateOptions := &core.GenerateBuildPlanOptions{
		RailpackVersion:          Version,
		BuildCommand:             cmd.String("build-cmd"),
		StartCommand:             cmd.String("start-cmd"),
		PreviousVersions:         previousVersions,
		ConfigFilePath:           cmd.String("config-file"),
		ErrorMissingStartCommand: cmd.Bool("error-missing-start"),
		Dev:                      true, // Force dev mode
	}

	buildResult := core.GenerateBuildPlan(app, env, generateOptions)

	return buildResult, app, env, nil
}

func extractDevConfig(buildResult *core.BuildResult) *DevConfigOutput {
	config := &DevConfigOutput{
		Variables: make(map[string]string),
	}

	// Extract detected language from metadata
	if provider, ok := buildResult.Metadata["providers"]; ok {
		config.DetectedLanguage = provider
	}
	// Also check DetectedProviders array
	if len(buildResult.DetectedProviders) > 0 {
		config.DetectedLanguage = buildResult.DetectedProviders[0]
	}

	// Extract deploy configuration
	if buildResult.Plan != nil {
		deploy := buildResult.Plan.Deploy

		// Extract start command
		if deploy.StartCmdHost != "" {
			config.StartCommandHost = deploy.StartCmdHost
		} else if deploy.StartCmd != "" {
			config.StartCommandHost = deploy.StartCmd
		}

		// Extract required port
		if deploy.RequiredPort != "" {
			config.RequiredPort = deploy.RequiredPort
		}

		// Extract variables
		if deploy.Variables != nil {
			config.Variables = deploy.Variables
		}

		// Extract APT packages from steps (collect all apt packages from all steps)
		var allAptPackages []string
		for _, step := range buildResult.Plan.Steps {
			// Look for any step that installs apt packages
			for _, cmd := range step.Commands {
				if execCmd, ok := cmd.(plan.ExecCommand); ok {
					cmdStr := execCmd.Cmd
					if strings.Contains(cmdStr, "apt-get install") {
						// Extract package names after apt-get install -y
						parts := strings.Split(cmdStr, "apt-get install -y ")
						if len(parts) > 1 {
							pkgStr := strings.TrimSpace(parts[1])
							pkgStr = strings.Trim(pkgStr, "'")
							// Split by spaces and collect individual packages
							packages := strings.Fields(pkgStr)
							allAptPackages = append(allAptPackages, packages...)
						}
					}
				}
			}
		}

		// Remove duplicates and join
		if len(allAptPackages) > 0 {
			// Use a map to remove duplicates
			uniquePackages := make(map[string]bool)
			var finalPackages []string
			for _, pkg := range allAptPackages {
				if !uniquePackages[pkg] {
					uniquePackages[pkg] = true
					finalPackages = append(finalPackages, pkg)
				}
			}
			config.AptPackages = strings.Join(finalPackages, ", ")
		}

		// Extract install command from steps
		for _, step := range buildResult.Plan.Steps {
			if step.Name == "install" {
				var installCommands []string
				for _, cmd := range step.Commands {
					if execCmd, ok := cmd.(plan.ExecCommand); ok {
						cmdStr := execCmd.Cmd

						// Node.js package managers
						if strings.Contains(cmdStr, "npm install") ||
							strings.Contains(cmdStr, "yarn install") ||
							strings.Contains(cmdStr, "pnpm install") ||
							strings.Contains(cmdStr, "bun install") {
							installCommands = append(installCommands, cmdStr)
						}

						// Python virtual env and pip
						if strings.Contains(cmdStr, "python -m venv") ||
							strings.Contains(cmdStr, "pip install") ||
							strings.Contains(cmdStr, "poetry install") ||
							strings.Contains(cmdStr, "pipenv install") ||
							strings.Contains(cmdStr, "pdm install") ||
							strings.Contains(cmdStr, "uv sync") {
							installCommands = append(installCommands, cmdStr)
						}

						// Ruby bundler
						if strings.Contains(cmdStr, "bundle install") {
							installCommands = append(installCommands, cmdStr)
						}

						// PHP composer
						if strings.Contains(cmdStr, "composer install") {
							installCommands = append(installCommands, cmdStr)
						}

						// Go (usually no install needed, but check for go mod download)
						if strings.Contains(cmdStr, "go mod download") ||
							strings.Contains(cmdStr, "go get") {
							installCommands = append(installCommands, cmdStr)
						}

						// Rust cargo
						if strings.Contains(cmdStr, "cargo build") {
							installCommands = append(installCommands, cmdStr)
						}
					}
				}

				// Join all install commands with " && "
				if len(installCommands) > 0 {
					config.InstallCommand = strings.Join(installCommands, " && ")

					// Replace container paths with local paths for development
					config.InstallCommand = strings.ReplaceAll(config.InstallCommand, "/app/.venv", ".venv")
					config.InstallCommand = strings.ReplaceAll(config.InstallCommand, "/app/", "./")
				}
				break
			}
		}
	}

	// Replace container paths with local paths in start command
	if config.StartCommandHost != "" {
		config.StartCommandHost = strings.ReplaceAll(config.StartCommandHost, "/app/.venv", ".venv")
		config.StartCommandHost = strings.ReplaceAll(config.StartCommandHost, "/app/", "./")
	}

	return config
}

func formatDevConfigPretty(config *DevConfigOutput) string {
	var output strings.Builder

	output.WriteString("Development Configuration:\n")
	output.WriteString("========================\n\n")
	output.WriteString(fmt.Sprintf("Detected Language: %s\n", config.DetectedLanguage))
	output.WriteString(fmt.Sprintf("Install Command: %s\n", config.InstallCommand))
	output.WriteString(fmt.Sprintf("Start Command: %s\n", config.StartCommandHost))
	output.WriteString(fmt.Sprintf("Required Port: %s\n", config.RequiredPort))

	if config.AptPackages != "" {
		output.WriteString(fmt.Sprintf("APT Packages: %s\n", config.AptPackages))
	}

	if len(config.Variables) > 0 {
		output.WriteString("\nEnvironment Variables:\n")
		for key, value := range config.Variables {
			output.WriteString(fmt.Sprintf("  %s=%s\n", key, value))
		}
	}

	return output.String()
}
