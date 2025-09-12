package python

import (
    "testing"

    testingUtils "github.com/railwayapp/railpack/core/testing"
    "github.com/stretchr/testify/require"
)

func TestPython_Django_Dev_UsesRunserver(t *testing.T) {
    ctx := testingUtils.CreateGenerateContext(t, "../../../examples/python-django")
    ctx.Dev = true

    provider := PythonProvider{}
    detected, err := provider.Detect(ctx)
    require.NoError(t, err)
    require.True(t, detected)

    err = provider.Initialize(ctx)
    require.NoError(t, err)

    err = provider.Plan(ctx)
    require.NoError(t, err)

    require.Contains(t, ctx.Deploy.StartCmd, "manage.py runserver")
}


