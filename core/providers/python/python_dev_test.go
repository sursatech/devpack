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
	require.Contains(t, ctx.Deploy.StartCmd, "0.0.0.0:8000")
    require.Equal(t, ctx.Deploy.StartCmd, ctx.Deploy.StartCmdHost) // Dev mode should use same command for both
    require.Equal(t, "8000", ctx.Deploy.RequiredPort)
}

func TestPython_Flask_Dev_UsesFlaskRun(t *testing.T) {
    ctx := testingUtils.CreateGenerateContext(t, "../../../examples/python-flask")
    ctx.Dev = true

    provider := PythonProvider{}
    detected, err := provider.Detect(ctx)
    require.NoError(t, err)
    require.True(t, detected)

    err = provider.Initialize(ctx)
    require.NoError(t, err)

    err = provider.Plan(ctx)
    require.NoError(t, err)

    require.Contains(t, ctx.Deploy.StartCmd, "flask")
    require.Contains(t, ctx.Deploy.StartCmd, "--host 0.0.0.0")
    require.Contains(t, ctx.Deploy.StartCmd, "--port 5000")
    require.Equal(t, ctx.Deploy.StartCmd, ctx.Deploy.StartCmdHost) // Dev mode should use same command for both
    require.Equal(t, "5000", ctx.Deploy.RequiredPort)
}

func TestPython_GetDevStartCommand(t *testing.T) {
    tests := []struct {
        name     string
        path     string
        expected string
    }{
        {
            name:     "Django project",
            path:     "../../../examples/python-django",
		expected: ".venv/bin/python manage.py runserver 0.0.0.0:8000",
        },
        {
            name:     "Flask project",
            path:     "../../../examples/python-flask",
            expected: ".venv/bin/flask --app main.py run --host 0.0.0.0 --port 5000",
        },
        {
            name:     "Poetry Flask project",
            path:     "../../../examples/python-poetry",
            expected: "poetry run flask --app main.py run --host 0.0.0.0 --port 5000",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := testingUtils.CreateGenerateContext(t, tt.path)
            provider := PythonProvider{}
            err := provider.Initialize(ctx)
            require.NoError(t, err)

            got := provider.GetDevStartCommand(ctx)
            require.Equal(t, tt.expected, got)
        })
    }
}

func TestPython_Poetry_Flask_Dev_UsesFlaskRun(t *testing.T) {
    ctx := testingUtils.CreateGenerateContext(t, "../../../examples/python-poetry")
    ctx.Dev = true

    provider := PythonProvider{}
    detected, err := provider.Detect(ctx)
    require.NoError(t, err)
    require.True(t, detected)

    err = provider.Initialize(ctx)
    require.NoError(t, err)

    err = provider.Plan(ctx)
    require.NoError(t, err)

    require.Contains(t, ctx.Deploy.StartCmd, "poetry run flask")
    require.Contains(t, ctx.Deploy.StartCmd, "--host 0.0.0.0")
    require.Contains(t, ctx.Deploy.StartCmd, "--port 5000")
    require.Equal(t, ctx.Deploy.StartCmd, ctx.Deploy.StartCmdHost) // Poetry uses same command for both
    require.Equal(t, "5000", ctx.Deploy.RequiredPort)
}

func TestPython_ProductionMode_NoRequiredPort(t *testing.T) {
    ctx := testingUtils.CreateGenerateContext(t, "../../../examples/python-django")
    ctx.Dev = false

    provider := PythonProvider{}
    err := provider.Initialize(ctx)
    require.NoError(t, err)

    err = provider.Plan(ctx)
    require.NoError(t, err)

    require.Empty(t, ctx.Deploy.RequiredPort) // Production mode should NOT have requiredPort
}


