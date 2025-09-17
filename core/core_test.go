package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/railwayapp/railpack/core/app"
	"github.com/railwayapp/railpack/core/logger"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	v := m.Run()
	snaps.Clean(m, snaps.CleanOpts{Sort: true})
	os.Exit(v)
}

// generate snapshot plan JSON for each build example and assert against it
func TestGenerateBuildPlanForExamples(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Get all the examples
	examplesDir := filepath.Join(filepath.Dir(wd), "examples")
	entries, err := os.ReadDir(examplesDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// For each example, generate a build plan that we can snapshot test
		t.Run(entry.Name(), func(t *testing.T) {
			examplePath := filepath.Join(examplesDir, entry.Name())

			userApp, err := app.NewApp(examplePath)
			require.NoError(t, err)

			env := app.NewEnvironment(nil)
			buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{})

			if !buildResult.Success {
				t.Fatalf("failed to generate build plan for %s: %s", entry.Name(), buildResult.Logs)
			}

			plan := buildResult.Plan

			// Remove the mise.toml asset since the versions may change between runs
			for _, step := range plan.Steps {
				for name := range step.Assets {
					if name == "mise.toml" {
						step.Assets[name] = "[mise.toml]"
					}
				}
			}

			snaps.MatchStandaloneJSON(t, plan)
		})
	}
}

func TestGenerateConfigFromFile_NotFound(t *testing.T) {
	// Use an existing example app directory so relative paths resolve
	appPath := "../examples/config-file"
	userApp, err := app.NewApp(appPath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	l := logger.NewLogger()

	options := &GenerateBuildPlanOptions{ConfigFilePath: "does-not-exist.railpack.json"}
	cfg, genErr := GenerateConfigFromFile(userApp, env, options, l)

	require.Error(t, genErr, "expected an error when explicit config file does not exist")
	require.Nil(t, cfg, "config should be nil on error")
}

func TestGenerateConfigFromFile_Malformed(t *testing.T) {
	appPath := "../examples/config-file"
	userApp, err := app.NewApp(appPath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	l := logger.NewLogger()

	options := &GenerateBuildPlanOptions{ConfigFilePath: "railpack.malformed.json"}
	cfg, genErr := GenerateConfigFromFile(userApp, env, options, l)

	require.Error(t, genErr, "expected an error for malformed JSON config file")
	require.Nil(t, cfg, "config should be nil on error")
}

func TestDevMode_NodeNext_UsesDevStart(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	examplePath := filepath.Join(filepath.Dir(wd), "examples", "node-next")

	userApp, err := app.NewApp(examplePath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{Dev: true})
	require.True(t, buildResult.Success)

	require.Equal(t, "npm run dev", buildResult.Plan.Deploy.StartCmd)
	// Next.js should expose full host-binding command via deploy.startCommandHost
	require.Equal(t, "npm run dev -- -H 0.0.0.0", buildResult.Plan.Deploy.StartCmdHost)
}

func TestDevMode_SPA_Vite_UsesDevScript_NoCaddy(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	examplePath := filepath.Join(filepath.Dir(wd), "examples", "node-vite-vanilla")

	userApp, err := app.NewApp(examplePath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{Dev: true})
	require.True(t, buildResult.Success)

	// start command should use dev
	require.Contains(t, buildResult.Plan.Deploy.StartCmd, "dev")

	// Vite should expose full host-binding command via deploy.startCommandHost
	require.Equal(t, "npm run dev -- --host", buildResult.Plan.Deploy.StartCmdHost)

	// ensure no caddy step present
	for _, step := range buildResult.Plan.Steps {
		require.NotEqual(t, "caddy", step.Name)
	}
}

func TestDevMode_NodeAngular_UsesStart_WithHost(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	examplePath := filepath.Join(filepath.Dir(wd), "examples", "node-angular")

	userApp, err := app.NewApp(examplePath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{Dev: true})
	require.True(t, buildResult.Success)

	// Angular example uses start script for dev
	require.Equal(t, "npm run start", buildResult.Plan.Deploy.StartCmd)
	require.Equal(t, "npm start -- --host 0.0.0.0", buildResult.Plan.Deploy.StartCmdHost)
}

func TestDevMode_Python_Django_UsesRunserver(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	examplePath := filepath.Join(filepath.Dir(wd), "examples", "python-django")

	userApp, err := app.NewApp(examplePath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{Dev: true})
	require.True(t, buildResult.Success)

	require.Contains(t, buildResult.Plan.Deploy.StartCmd, "manage.py runserver")
}

func TestDevMode_Deno_UsesTaskDev(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	examplePath := filepath.Join(filepath.Dir(wd), "examples", "deno-2")

	userApp, err := app.NewApp(examplePath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{Dev: true})
	require.True(t, buildResult.Success)

	require.Equal(t, "deno task dev", buildResult.Plan.Deploy.StartCmd)
}

func TestDevMode_Golang_GoRun(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	examplePath := filepath.Join(filepath.Dir(wd), "examples", "go-mod")

	userApp, err := app.NewApp(examplePath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{Dev: true})
	require.True(t, buildResult.Success)

	require.Contains(t, buildResult.Plan.Deploy.StartCmd, "go run")
}

func TestDevMode_Java_Gradle_Run(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	examplePath := filepath.Join(filepath.Dir(wd), "examples", "java-gradle")

	userApp, err := app.NewApp(examplePath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{Dev: true})
	require.True(t, buildResult.Success)

	require.Contains(t, buildResult.Plan.Deploy.StartCmd, "gradle")
	require.Contains(t, buildResult.Plan.Deploy.StartCmd, "run")
}

func TestDevMode_PHP_Laravel_Serve(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	examplePath := filepath.Join(filepath.Dir(wd), "examples", "php-laravel-12-react")

	userApp, err := app.NewApp(examplePath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{Dev: true})
	require.True(t, buildResult.Success)

	require.Contains(t, buildResult.Plan.Deploy.StartCmd, "php artisan serve")
}

func TestDevMode_Rust_UsesCargoRun(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	examplePath := filepath.Join(filepath.Dir(wd), "examples", "rust-rocket")

	userApp, err := app.NewApp(examplePath)
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	buildResult := GenerateBuildPlan(userApp, env, &GenerateBuildPlanOptions{Dev: true})
	require.True(t, buildResult.Success)

	require.Equal(t, "cargo run", buildResult.Plan.Deploy.StartCmd)
	require.Equal(t, "cargo run", buildResult.Plan.Deploy.StartCmdHost)
	
	// Check development environment variables
	require.Equal(t, "0.0.0.0", buildResult.Plan.Deploy.Variables["ROCKET_ADDRESS"])
	require.Equal(t, "development", buildResult.Plan.Deploy.Variables["ROCKET_ENV"])
	require.Equal(t, "debug", buildResult.Plan.Deploy.Variables["ROCKET_LOG_LEVEL"])
	require.Equal(t, "debug", buildResult.Plan.Deploy.Variables["RUST_LOG"])
}
