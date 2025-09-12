package golang

import (
    "testing"

    testingUtils "github.com/railwayapp/railpack/core/testing"
    "github.com/stretchr/testify/require"
)

func TestGo_Dev_UsesGoRun(t *testing.T) {
    ctx := testingUtils.CreateGenerateContext(t, "../../../examples/go-mod")
    ctx.Dev = true

    provider := GoProvider{}
    detected, err := provider.Detect(ctx)
    require.NoError(t, err)
    require.True(t, detected)

    err = provider.Initialize(ctx)
    require.NoError(t, err)

    err = provider.Plan(ctx)
    require.NoError(t, err)

    require.Contains(t, ctx.Deploy.StartCmd, "go run")
}


