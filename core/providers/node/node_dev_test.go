package node

import (
	"testing"

	"github.com/railwayapp/railpack/core/app"
	"github.com/railwayapp/railpack/core/config"
	"github.com/railwayapp/railpack/core/generate"
	"github.com/railwayapp/railpack/core/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNode_Dev_HasRequiredPort(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Next.js app",
			path:     "../../../examples/node-next",
			expected: "3000",
		},
		{
			name:     "Angular app",
			path:     "../../../examples/node-angular",
			expected: "4200",
		},
		{
			name:     "Vite app",
			path:     "../../../examples/node-vite-react",
			expected: "5173",
		},
		{
			name:     "Astro app",
			path:     "../../../examples/node-astro",
			expected: "4321",
		},
		{
			name:     "React Router app",
			path:     "../../../examples/node-vite-react-router-spa",
			expected: "5173",
		},
		{
			name:     "CRA app",
			path:     "../../../examples/node-cra",
			expected: "3000",
		},
		{
			name:     "Nuxt app",
			path:     "../../../examples/node-nuxt",
			expected: "3000",
		},
		{
			name:     "Remix app",
			path:     "../../../examples/node-remix",
			expected: "3000",
		},
		{
			name:     "Tanstack Start app",
			path:     "../../../examples/node-tanstack-start",
			expected: "3000",
		},
		{
			name:     "Basic Node.js API",
			path:     "../../../examples/node-npm",
			expected: "3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userApp, err := app.NewApp(tt.path)
			require.NoError(t, err)

			provider := &NodeProvider{}
			err = provider.Initialize(&generate.GenerateContext{App: userApp})
			require.NoError(t, err)

			env := app.NewEnvironment(nil)
			config := config.EmptyConfig()
			ctx, err := generate.NewGenerateContext(userApp, env, config, logger.NewLogger())
			require.NoError(t, err)
			ctx.Dev = true

			port := provider.getDevPort(ctx)
			assert.Equal(t, tt.expected, port)
		})
	}
}

func TestNode_Dev_NoRequiredPortInProduction(t *testing.T) {
	userApp, err := app.NewApp("../../../examples/node-next")
	require.NoError(t, err)

	provider := &NodeProvider{}
	err = provider.Initialize(&generate.GenerateContext{App: userApp})
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	config := config.EmptyConfig()
	ctx, err := generate.NewGenerateContext(userApp, env, config, logger.NewLogger())
	require.NoError(t, err)
	ctx.Dev = false // Production mode

	port := provider.getDevPort(ctx)
	assert.Equal(t, "3000", port) // Should still return default port for production
}

func TestNode_Dev_FrameworkDetection(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Next.js",
			path:     "../../../examples/node-next",
			expected: "3000",
		},
		{
			name:     "Angular SPA",
			path:     "../../../examples/node-angular",
			expected: "4200",
		},
		{
			name:     "Vite SPA",
			path:     "../../../examples/node-vite-react",
			expected: "5173",
		},
		{
			name:     "Astro SPA",
			path:     "../../../examples/node-astro",
			expected: "4321",
		},
		{
			name:     "Basic Node.js API",
			path:     "../../../examples/node-npm",
			expected: "3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userApp, err := app.NewApp(tt.path)
			require.NoError(t, err)

			provider := &NodeProvider{}
			err = provider.Initialize(&generate.GenerateContext{App: userApp})
			require.NoError(t, err)

			env := app.NewEnvironment(nil)
			config := config.EmptyConfig()
			ctx, err := generate.NewGenerateContext(userApp, env, config, logger.NewLogger())
			require.NoError(t, err)
			ctx.Dev = true

			port := provider.getDevPort(ctx)
			assert.Equal(t, tt.expected, port)
		})
	}
}

func TestNode_Dev_FrameworkSpecificEnvVars(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedEnvVar string
		expectedValue  string
	}{
		{
			name:           "Vite - VITE allowed hosts",
			path:           "../../../examples/node-vite-react",
			expectedEnvVar: "__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS",
			expectedValue:  ".sursakit.app",
		},
		{
			name:           "React Router - VITE allowed hosts",
			path:           "../../../examples/node-vite-react-router-spa",
			expectedEnvVar: "__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS",
			expectedValue:  ".sursakit.app",
		},
		{
			name:           "CRA - HOST",
			path:           "../../../examples/node-cra",
			expectedEnvVar: "HOST",
			expectedValue:  "0.0.0.0",
		},
		{
			name:           "Angular - NG_CLI_ANALYTICS",
			path:           "../../../examples/node-angular",
			expectedEnvVar: "NG_CLI_ANALYTICS",
			expectedValue:  "false",
		},
		{
			name:           "Next.js - HOSTNAME",
			path:           "../../../examples/node-next",
			expectedEnvVar: "HOSTNAME",
			expectedValue:  "0.0.0.0",
		},
		{
			name:           "Nuxt - NUXT_HOST",
			path:           "../../../examples/node-nuxt",
			expectedEnvVar: "NUXT_HOST",
			expectedValue:  "0.0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userApp, err := app.NewApp(tt.path)
			require.NoError(t, err)

			provider := &NodeProvider{}
			err = provider.Initialize(&generate.GenerateContext{App: userApp})
			require.NoError(t, err)

			env := app.NewEnvironment(nil)
			config := config.EmptyConfig()
			ctx, err := generate.NewGenerateContext(userApp, env, config, logger.NewLogger())
			require.NoError(t, err)
			ctx.Dev = true

			envVars := provider.GetNodeEnvVars(ctx)
			assert.Equal(t, tt.expectedValue, envVars[tt.expectedEnvVar])
		})
	}
}

func TestNode_Dev_NoFrameworkEnvVarsInProduction(t *testing.T) {
	userApp, err := app.NewApp("../../../examples/node-vite-react")
	require.NoError(t, err)

	provider := &NodeProvider{}
	err = provider.Initialize(&generate.GenerateContext{App: userApp})
	require.NoError(t, err)

	env := app.NewEnvironment(nil)
	config := config.EmptyConfig()
	ctx, err := generate.NewGenerateContext(userApp, env, config, logger.NewLogger())
	require.NoError(t, err)
	ctx.Dev = false // Production mode

	envVars := provider.GetNodeEnvVars(ctx)

	// Framework-specific environment variables should not be set in production
	assert.Empty(t, envVars["__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS"])
	assert.Empty(t, envVars["HOST"])
	assert.Empty(t, envVars["NG_CLI_ANALYTICS"])
	assert.Empty(t, envVars["HOSTNAME"])
	assert.Empty(t, envVars["NUXT_HOST"])
}
