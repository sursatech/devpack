package node

import (
	"testing"

	testingUtils "github.com/railwayapp/railpack/core/testing"
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
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			ctx.Dev = true // Enable dev mode

			provider := NodeProvider{}
			err := provider.Initialize(ctx)
			require.NoError(t, err)

			err = provider.Plan(ctx)
			require.NoError(t, err)

			require.Equal(t, tt.expected, ctx.Deploy.RequiredPort)
		})
	}
}

func TestNode_Dev_NoRequiredPortInProduction(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/node-next")
	ctx.Dev = false // Production mode

	provider := NodeProvider{}
	err := provider.Initialize(ctx)
	require.NoError(t, err)

	err = provider.Plan(ctx)
	require.NoError(t, err)

	require.Empty(t, ctx.Deploy.RequiredPort)
}

func TestNode_Dev_FrameworkDetection(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		expectedPort    string
		expectedSPA     bool
		expectedRuntime string
	}{
		{
			name:            "Next.js",
			path:            "../../../examples/node-next",
			expectedPort:    "3000",
			expectedSPA:     false,
			expectedRuntime: "next",
		},
		{
			name:            "Angular SPA",
			path:            "../../../examples/node-angular",
			expectedPort:    "4200",
			expectedSPA:     true,
			expectedRuntime: "angular",
		},
		{
			name:            "Vite SPA",
			path:            "../../../examples/node-vite-react",
			expectedPort:    "5173",
			expectedSPA:     true,
			expectedRuntime: "vite",
		},
		{
			name:            "Astro SPA",
			path:            "../../../examples/node-astro",
			expectedPort:    "4321",
			expectedSPA:     true,
			expectedRuntime: "astro",
		},
		{
			name:            "Basic Node.js API",
			path:            "../../../examples/node-npm",
			expectedPort:    "3000",
			expectedSPA:     false,
			expectedRuntime: "node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			ctx.Dev = true

			provider := NodeProvider{}
			err := provider.Initialize(ctx)
			require.NoError(t, err)

			err = provider.Plan(ctx)
			require.NoError(t, err)

			require.Equal(t, tt.expectedPort, ctx.Deploy.RequiredPort)
			require.Equal(t, tt.expectedSPA, ctx.Metadata.Get("nodeIsSPA") == "true")
			require.Equal(t, tt.expectedRuntime, ctx.Metadata.Get("nodeRuntime"))
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
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			ctx.Dev = true

			provider := NodeProvider{}
			err := provider.Initialize(ctx)
			require.NoError(t, err)

			envVars := provider.GetNodeEnvVars(ctx)
			require.Equal(t, tt.expectedValue, envVars[tt.expectedEnvVar])
		})
	}
}

func TestNode_Dev_NoFrameworkEnvVarsInProduction(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/node-vite-react")
	ctx.Dev = false // Production mode

	provider := NodeProvider{}
	err := provider.Initialize(ctx)
	require.NoError(t, err)

	envVars := provider.GetNodeEnvVars(ctx)

	// These framework-specific env vars should not be set in production
	require.Empty(t, envVars["__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS"])
	require.Empty(t, envVars["HOST"])
	require.Empty(t, envVars["NG_CLI_ANALYTICS"])
	require.Empty(t, envVars["HOSTNAME"])
	require.Empty(t, envVars["NUXT_HOST"])
}
