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


