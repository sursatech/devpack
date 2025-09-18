package staticfile

import (
	"testing"

	testingUtils "github.com/railwayapp/railpack/core/testing"
	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "index",
			path: "../../../examples/staticfile-index",
			want: true,
		},
		{
			name: "config",
			path: "../../../examples/staticfile-config",
			want: true,
		},
		{
			name: "npm",
			path: "../../../examples/node-npm",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			provider := StaticfileProvider{}
			got, err := provider.Detect(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGetRootDir(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		envVars     map[string]string
		want        string
		expectError bool
	}{
		{
			name: "from env var",
			path: "../../../examples/staticfile-index",
			envVars: map[string]string{
				"RAILPACK_STATIC_FILE_ROOT": "/custom/path",
			},
			want:        "/custom/path",
			expectError: false,
		},
		{
			name:        "from staticfile config",
			path:        "../../../examples/staticfile-config",
			envVars:     map[string]string{},
			want:        "hello",
			expectError: false,
		},
		{
			name:        "from index.html",
			path:        "../../../examples/staticfile-index",
			envVars:     map[string]string{},
			want:        ".",
			expectError: false,
		},
		{
			name:        "no root dir",
			path:        "../../../examples/node-npm",
			envVars:     map[string]string{},
			want:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			for k, v := range tt.envVars {
				ctx.Env.SetVariable(k, v)
			}

			got, err := getRootDir(ctx)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetDevStartCommand(t *testing.T) {
	tests := []struct {
		name     string
		rootDir  string
		expected string
	}{
		{
			name:     "root directory",
			rootDir:  ".",
			expected: "lite-server",
		},
		{
			name:     "public directory",
			rootDir:  "public",
			expected: "lite-server",
		},
		{
			name:     "custom directory",
			rootDir:  "dist",
			expected: "lite-server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := StaticfileProvider{RootDir: tt.rootDir}
			ctx := testingUtils.CreateGenerateContext(t, "../../../examples/staticfile-index")
			
			got := provider.GetDevStartCommand(ctx)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestDevMode_UsesNodeServer(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/staticfile-index")
	ctx.Dev = true
	
	provider := StaticfileProvider{}
	err := provider.Initialize(ctx)
	require.NoError(t, err)
	
	err = provider.Plan(ctx)
	require.NoError(t, err)
	
	// Should use Node.js server in dev mode
	require.Equal(t, "lite-server", ctx.Deploy.StartCmd)
	require.Equal(t, "lite-server", ctx.Deploy.StartCmdHost)
	require.Equal(t, "3000", ctx.Deploy.RequiredPort) // Development mode should have requiredPort (lite-server default)
}

func TestProductionMode_UsesCaddy(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/staticfile-index")
	ctx.Dev = false
	
	provider := StaticfileProvider{}
	err := provider.Initialize(ctx)
	require.NoError(t, err)
	
	err = provider.Plan(ctx)
	require.NoError(t, err)
	
	// Should use Caddy in production mode
	require.Contains(t, ctx.Deploy.StartCmd, "caddy run")
	require.Contains(t, ctx.Deploy.StartCmd, "--config Caddyfile")
	require.Empty(t, ctx.Deploy.RequiredPort) // Production mode should NOT have requiredPort
}
