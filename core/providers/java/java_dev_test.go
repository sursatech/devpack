package java

import (
    "testing"

    testingUtils "github.com/railwayapp/railpack/core/testing"
    "github.com/stretchr/testify/require"
)

func TestJava_Dev_UsesRunTask(t *testing.T) {
    ctx := testingUtils.CreateGenerateContext(t, "../../../examples/java-gradle")
    ctx.Dev = true

    provider := JavaProvider{}
    detected, err := provider.Detect(ctx)
    require.NoError(t, err)
    require.True(t, detected)

    err = provider.Initialize(ctx)
    require.NoError(t, err)

    err = provider.Plan(ctx)
    require.NoError(t, err)

    require.Contains(t, ctx.Deploy.StartCmd, "run")
}


