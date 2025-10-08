package rust

import (
	"testing"

	testingUtils "github.com/railwayapp/railpack/core/testing"
	"github.com/stretchr/testify/require"
)

func TestRust_Dev_UsesCargoRun(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/rust-rocket")
	ctx.Dev = true

	provider := RustProvider{}
	detected, err := provider.Detect(ctx)
	require.NoError(t, err)
	require.True(t, detected)

	err = provider.Initialize(ctx)
	require.NoError(t, err)

	err = provider.Plan(ctx)
	require.NoError(t, err)

	// In development mode, should use cargo run with port as CLI args
	require.Equal(t, "cargo run -- --port 8000 --address 0.0.0.0", ctx.Deploy.StartCmd)

	// Should have StartCmdHost for development mode
	require.Equal(t, "cargo run -- --port 8000 --address 0.0.0.0", ctx.Deploy.StartCmdHost)

	// Should have development environment variables
	require.Equal(t, "0.0.0.0", ctx.Deploy.Variables["ROCKET_ADDRESS"])
	require.Equal(t, "development", ctx.Deploy.Variables["ROCKET_ENV"])
	require.Equal(t, "debug", ctx.Deploy.Variables["ROCKET_LOG_LEVEL"])
	require.Equal(t, "debug", ctx.Deploy.Variables["RUST_LOG"])
	require.Equal(t, "8000", ctx.Deploy.Variables["ROCKET_PORT"])
}

func TestRust_Production_UsesBinary(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/rust-rocket")
	ctx.Dev = false

	provider := RustProvider{}
	detected, err := provider.Detect(ctx)
	require.NoError(t, err)
	require.True(t, detected)

	err = provider.Initialize(ctx)
	require.NoError(t, err)

	err = provider.Plan(ctx)
	require.NoError(t, err)

	// In production mode, should use pre-built binary
	require.Equal(t, "./bin/rocket", ctx.Deploy.StartCmd)

	// Should not have StartCmdHost in production
	require.Empty(t, ctx.Deploy.StartCmdHost)

	// Should only have basic environment variables
	require.Equal(t, "0.0.0.0", ctx.Deploy.Variables["ROCKET_ADDRESS"])
	require.Empty(t, ctx.Deploy.Variables["ROCKET_ENV"])
	require.Empty(t, ctx.Deploy.Variables["ROCKET_LOG_LEVEL"])
	require.Empty(t, ctx.Deploy.Variables["RUST_LOG"])
}

func TestRust_GetDevStartCommand(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/rust-rocket")
	ctx.Dev = true

	provider := RustProvider{}
	devCmd := provider.GetDevStartCommand(ctx)

	require.Equal(t, "cargo run -- --port 8000 --address 0.0.0.0", devCmd)
}

func TestRust_GetRustEnvVars_Dev(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/rust-rocket")
	ctx.Dev = true

	provider := RustProvider{}
	envVars := provider.GetRustEnvVars(ctx)

	require.Equal(t, "0.0.0.0", envVars["ROCKET_ADDRESS"])
	require.Equal(t, "development", envVars["ROCKET_ENV"])
	require.Equal(t, "debug", envVars["ROCKET_LOG_LEVEL"])
	require.Equal(t, "debug", envVars["RUST_LOG"])
	require.Equal(t, "8000", envVars["ROCKET_PORT"])
}

func TestRust_GetRustEnvVars_Production(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/rust-rocket")
	ctx.Dev = false

	provider := RustProvider{}
	envVars := provider.GetRustEnvVars(ctx)

	require.Equal(t, "0.0.0.0", envVars["ROCKET_ADDRESS"])
	require.Empty(t, envVars["ROCKET_ENV"])
	require.Empty(t, envVars["ROCKET_LOG_LEVEL"])
	require.Empty(t, envVars["RUST_LOG"])
}
