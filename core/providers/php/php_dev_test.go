package php

import (
	"testing"

	testingUtils "github.com/railwayapp/railpack/core/testing"
	"github.com/stretchr/testify/require"
)

func TestPHP_Laravel_Dev_UsesArtisanServe(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/php-laravel-12-react")
	ctx.Dev = true

	provider := PhpProvider{}
	detected, err := provider.Detect(ctx)
	require.NoError(t, err)
	require.True(t, detected)

	err = provider.Initialize(ctx)
	require.NoError(t, err)

	err = provider.Plan(ctx)
	require.NoError(t, err)

	require.Contains(t, ctx.Deploy.StartCmd, "php artisan serve")
}

func TestPHP_Dev_Laravel_HasStartCmdHost(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/php-laravel-12-react")
	ctx.Dev = true

	provider := PhpProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	require.Contains(t, ctx.Deploy.StartCmdHost, "php artisan serve")
	require.Contains(t, ctx.Deploy.StartCmdHost, "--port 8000")
}

func TestPHP_Dev_Vanilla_HasStartCmdHost(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/php-vanilla")
	ctx.Dev = true

	provider := PhpProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	require.Contains(t, ctx.Deploy.StartCmdHost, "php -S 0.0.0.0:8000")
}

func TestPHP_Dev_HasRequiredPort(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/php-laravel-12-react")
	ctx.Dev = true

	provider := PhpProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	require.Equal(t, "8000", ctx.Deploy.RequiredPort)
}

func TestPHP_Dev_HasDevEnvVars(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/php-laravel-12-react")
	ctx.Dev = true

	provider := PhpProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	require.Equal(t, "local", ctx.Deploy.Variables["APP_ENV"])
	require.Equal(t, "true", ctx.Deploy.Variables["APP_DEBUG"])
	require.Equal(t, "debug", ctx.Deploy.Variables["LOG_LEVEL"])
}

func TestPHP_Prod_HasProdEnvVars(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/php-laravel-12-react")
	ctx.Dev = false

	provider := PhpProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	require.Equal(t, "production", ctx.Deploy.Variables["APP_ENV"])
	require.Equal(t, "false", ctx.Deploy.Variables["APP_DEBUG"])
	require.Equal(t, "error", ctx.Deploy.Variables["LOG_LEVEL"])
}

func TestPHP_Dev_Laravel_HasLaravelDevVars(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/php-laravel-12-react")
	ctx.Dev = true

	provider := PhpProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	require.Equal(t, "sqlite", ctx.Deploy.Variables["DB_CONNECTION"])
	require.Equal(t, ":memory:", ctx.Deploy.Variables["DB_DATABASE"])
	require.Equal(t, "array", ctx.Deploy.Variables["CACHE_DRIVER"])
	require.Equal(t, "array", ctx.Deploy.Variables["SESSION_DRIVER"])
}

func TestPHP_Prod_Laravel_HasLaravelProdVars(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/php-laravel-12-react")
	ctx.Dev = false

	provider := PhpProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	require.Equal(t, "file", ctx.Deploy.Variables["CACHE_DRIVER"])
	require.Equal(t, "file", ctx.Deploy.Variables["SESSION_DRIVER"])
	require.Equal(t, "smtp", ctx.Deploy.Variables["MAIL_MAILER"])
}

func TestPHP_Framework_Detection(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		isLaravel     bool
		isSymfony     bool
		isCodeIgniter bool
	}{
		{
			name:          "Laravel project",
			path:          "../../../examples/php-laravel-12-react",
			isLaravel:     true,
			isSymfony:     false,
			isCodeIgniter: false,
		},
		{
			name:          "Vanilla PHP project",
			path:          "../../../examples/php-vanilla",
			isLaravel:     false,
			isSymfony:     false,
			isCodeIgniter: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testingUtils.CreateGenerateContext(t, tt.path)
			provider := PhpProvider{}

			require.Equal(t, tt.isLaravel, provider.usesLaravel(ctx))
			require.Equal(t, tt.isSymfony, provider.usesSymfony(ctx))
			require.Equal(t, tt.isCodeIgniter, provider.usesCodeIgniter(ctx))
		})
	}
}
