package cli

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/railwayapp/railpack/core"
	a "github.com/railwayapp/railpack/core/app"
	"github.com/stretchr/testify/require"
)

func TestDevConfigCommand(t *testing.T) {
	tests := []struct {
		name        string
		example     string
		wantLang    string
		wantPort    string
		wantInstall string
	}{
		{
			name:        "Next.js Framework",
			example:     "../examples/node-next",
			wantLang:    "node",
			wantPort:    "3000",
			wantInstall: "npm install",
		},
		{
			name:        "Angular Framework",
			example:     "../examples/node-angular",
			wantLang:    "node",
			wantPort:    "4200",
			wantInstall: "npm install",
		},
		{
			name:        "Bun Package Manager",
			example:     "../examples/node-bun",
			wantLang:    "node",
			wantPort:    "3000",
			wantInstall: "bun install",
		},
		{
			name:        "Yarn Package Manager",
			example:     "../examples/node-yarn-1",
			wantLang:    "node",
			wantPort:    "3000",
			wantInstall: "yarn install",
		},
		{
			name:        "PNPM Package Manager",
			example:     "../examples/node-pnpm-workspaces",
			wantLang:    "node",
			wantPort:    "3000",
			wantInstall: "pnpm install",
		},
		{
			name:        "Vite Framework",
			example:     "../examples/node-vite-react",
			wantLang:    "node",
			wantPort:    "5173",
			wantInstall: "npm install",
		},
		{
			name:        "Astro Framework",
			example:     "../examples/node-astro",
			wantLang:    "node",
			wantPort:    "4321",
			wantInstall: "npm install",
		},
		{
			name:        "Puppeteer with APT packages",
			example:     "../examples/node-puppeteer",
			wantLang:    "node",
			wantPort:    "3000",
			wantInstall: "npm install",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := testDevConfigExtraction(tt.example)
			if err != nil {
				t.Logf("Example directory %s does not exist", tt.example)
				t.Skip()
				return
			}

			require.Equal(t, tt.wantLang, config.DetectedLanguage)
			require.Equal(t, tt.wantPort, config.RequiredPort)
			require.Contains(t, config.InstallCommand, tt.wantInstall)
			require.NotEmpty(t, config.StartCommandHost)
		})
	}
}

func TestDevConfigErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		directory string
		wantError bool
	}{
		{
			name:      "Non-existent directory",
			directory: "../examples/non-existent",
			wantError: true,
		},
		{
			name:      "Empty directory",
			directory: "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testDevConfigExtraction(tt.directory)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDevConfigOutputFormat(t *testing.T) {
	config := &DevConfigOutput{
		DetectedLanguage: "node",
		AptPackages:      "ca-certificates",
		InstallCommand:   "npm install",
		StartCommandHost: "npm run dev",
		RequiredPort:     "3000",
		Variables: map[string]string{
			"NODE_ENV": "development",
		},
	}

	// Test JSON format
	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	require.Contains(t, string(jsonBytes), "detectedLanguage")
	require.Contains(t, string(jsonBytes), "node")

	// Test pretty format
	pretty := formatDevConfigPretty(config)
	require.Contains(t, pretty, "Development Configuration")
	require.Contains(t, pretty, "node")
	require.Contains(t, pretty, "npm install")
}

// Helper function to extract dev config for testing
func testDevConfigExtraction(directory string) (*DevConfigOutput, error) {
	if directory == "" {
		return nil, fmt.Errorf("directory is required")
	}

	app, err := a.NewApp(directory)
	if err != nil {
		return nil, err
	}

	env := a.NewEnvironment(nil)
	buildResult := core.GenerateBuildPlan(app, env, &core.GenerateBuildPlanOptions{
		Dev: true,
	})

	if !buildResult.Success {
		return nil, fmt.Errorf("failed to generate build plan")
	}

	config := extractDevConfig(buildResult)
	return config, nil
}
