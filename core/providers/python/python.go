package python

import (
	"fmt"
	"maps"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/railwayapp/railpack/core/generate"
	"github.com/railwayapp/railpack/core/plan"
	"github.com/railwayapp/railpack/internal/utils"
)

const (
	DEFAULT_PYTHON_VERSION = "3.11"
	UV_CACHE_DIR           = "/opt/uv-cache"
	PIP_CACHE_DIR          = "/opt/pip-cache"
	VENV_PATH              = "/app/.venv"
	LOCAL_BIN_PATH         = "/root/.local/bin"
)

// GetVenvPath returns the appropriate virtual environment path based on context
func (p *PythonProvider) GetVenvPath(ctx *generate.GenerateContext) string {
	// Always use /app/.venv for containerized environments
	return VENV_PATH
}

// GetVenvPathForHost returns the virtual environment path for host/local development
func (p *PythonProvider) GetVenvPathForHost(ctx *generate.GenerateContext) string {
	// Always use relative path for host/local development
	return ".venv"
}

// GetVenvPathForInstall returns the appropriate virtual environment path for install commands
func (p *PythonProvider) GetVenvPathForInstall(ctx *generate.GenerateContext) string {
	// For install commands, always use .venv (relative path) for local development
	// This ensures install commands work on local machines
	return ".venv"
}

type PythonProvider struct{}

func (p *PythonProvider) Name() string {
	return "python"
}

func (p *PythonProvider) Initialize(ctx *generate.GenerateContext) error {
	return nil
}

func (p *PythonProvider) Detect(ctx *generate.GenerateContext) (bool, error) {
	hasPython := ctx.App.HasMatch("main.py") ||
		p.hasRequirements(ctx) ||
		p.hasPyproject(ctx) ||
		p.hasPipfile(ctx)

	return hasPython, nil
}

func (p *PythonProvider) Plan(ctx *generate.GenerateContext) error {
	p.InstallMisePackages(ctx, ctx.GetMiseStepBuilder())

	install := ctx.NewCommandStep("install")
	install.AddInput(plan.NewStepLayer(p.GetBuilderDeps(ctx).Name()))

	install.Secrets = []string{}
	install.UseSecretsWithPrefixes([]string{"PYTHON", "PIP", "PIPX", "UV", "PDM", "POETRY"})

	build := ctx.NewCommandStep("build")
	installOutputs := []string{}

	if p.hasRequirements(ctx) {
		installOutputs = p.InstallPip(ctx, install)
	} else if p.hasPyproject(ctx) && p.hasUv(ctx) {
		installOutputs = p.InstallUv(ctx, install)
		build.AddCommands([]plan.Command{
			// the project is not installed during the install phase, because it requires the project source
			plan.NewExecCommand("uv sync --locked --no-dev --no-editable"),
		})
	} else if p.hasPyproject(ctx) && p.hasPoetry(ctx) {
		installOutputs = p.InstallPoetry(ctx, install)
	} else if p.hasPyproject(ctx) && p.hasPdm(ctx) {
		installOutputs = p.InstallPDM(ctx, install)
	} else if p.hasPipfile(ctx) {
		installOutputs = p.InstallPipenv(ctx, install)
	}

	p.addMetadata(ctx)

	build.AddInput(plan.NewStepLayer(install.Name()))
	build.AddInput(plan.NewLocalLayer())

	ctx.Deploy.StartCmd = p.GetStartCommand(ctx)
	
	// Use different environment variables for dev vs production
	if ctx.Dev {
		maps.Copy(ctx.Deploy.Variables, p.GetPythonDevEnvVars(ctx))
	} else {
		maps.Copy(ctx.Deploy.Variables, p.GetPythonProdEnvVars(ctx))
	}
	
	// Add virtual environment to PATH for deploy step
	if ctx.Deploy.Paths == nil {
		ctx.Deploy.Paths = []string{}
	}
	// Use the same venv path as install commands for consistency
	venvPath := p.GetVenvPathForInstall(ctx)
	ctx.Deploy.Paths = append(ctx.Deploy.Paths, venvPath+"/bin")

	// In dev mode, prefer a development server command when available
	if ctx.Dev {
		if devCmd := p.GetDevStartCommand(ctx); devCmd != "" {
			ctx.Deploy.StartCmd = devCmd
			ctx.Deploy.StartCmdHost = p.GetDevStartCommandHost(ctx)
			ctx.Deploy.RequiredPort = p.getDevPort(ctx) // Get framework-specific port
		}
	}

	installArtifacts := plan.NewStepLayer(build.Name(), plan.Filter{
		Include: installOutputs,
	})

	p.AddRuntimeDeps(ctx)

	ctx.Deploy.AddInputs([]plan.Layer{
		ctx.GetMiseStepBuilder().GetLayer(),
		installArtifacts,
		plan.NewStepLayer(build.Name(), plan.Filter{
			Include: []string{"."},
			Exclude: []string{strings.TrimPrefix(venvPath, "/app/")},
		}),
	})

	return nil
}

func (p *PythonProvider) GetStartCommand(ctx *generate.GenerateContext) string {
	startCommand := ""
	hasPoetry := p.hasPoetry(ctx)

	if p.isDjango(ctx) {
		startCommand = p.getDjangoStartCommand(ctx)
	}

	mainPythonFile := p.getMainPythonFile(ctx)
	hasMainPythonFile := mainPythonFile != ""

	if p.isFasthtml(ctx) && hasMainPythonFile && p.usesDep(ctx, "uvicorn") {
		if hasPoetry {
			startCommand = "poetry run uvicorn main:app --host 0.0.0.0 --port ${PORT:-8000}"
		} else {
			startCommand = "uvicorn main:app --host 0.0.0.0 --port ${PORT:-8000}"
		}
	}

	if p.isFlask(ctx) && hasMainPythonFile && p.usesDep(ctx, "gunicorn") {
		if hasPoetry {
			startCommand = "poetry run gunicorn --bind 0.0.0.0:${PORT:-8000} main:app"
		} else {
			startCommand = "gunicorn --bind 0.0.0.0:${PORT:-8000} main:app"
		}
	}

	if startCommand == "" && hasMainPythonFile {
		if hasPoetry {
			startCommand = fmt.Sprintf("poetry run python %s", mainPythonFile)
		} else {
			venvPath := p.GetVenvPath(ctx)
			startCommand = fmt.Sprintf("%s/bin/python %s", venvPath, mainPythonFile)
		}
	}

	return startCommand
}

// GetDevStartCommand returns a development-friendly start command
func (p *PythonProvider) GetDevStartCommand(ctx *generate.GenerateContext) string {
	// Check if this is a Poetry project
	hasPoetry := p.hasPoetry(ctx)
	
	// Django: use runserver with development setup
	if p.isDjango(ctx) {
		if hasPoetry {
			return "poetry run python manage.py runserver 0.0.0.0:8000"
		}
		venvPath := p.GetVenvPath(ctx)
		return fmt.Sprintf("%s/bin/python manage.py runserver 0.0.0.0:8000", venvPath)
	}

    mainPythonFile := p.getMainPythonFile(ctx)
    hasMainPythonFile := mainPythonFile != ""

    // FastAPI (uvicorn) with reload if available
    if p.isFastAPI(ctx) && hasMainPythonFile {
        if hasPoetry {
            return "poetry run uvicorn main:app --reload --host 0.0.0.0 --port 8000"
        }
        venvPath := p.GetVenvPath(ctx)
        return fmt.Sprintf("%s/bin/uvicorn main:app --reload --host 0.0.0.0 --port 8000", venvPath)
    }

    // Streamlit apps
    if p.isStreamlit(ctx) {
        if hasPoetry {
            return "poetry run streamlit run main.py --server.address 0.0.0.0 --server.port 8501"
        }
        venvPath := p.GetVenvPath(ctx)
        return fmt.Sprintf("%s/bin/streamlit run main.py --server.address 0.0.0.0 --server.port 8501", venvPath)
    }

    // Gradio apps
    if p.isGradio(ctx) {
        if hasPoetry {
            return "poetry run python main.py --server-name 0.0.0.0 --server-port 7860"
        }
        venvPath := p.GetVenvPath(ctx)
        return fmt.Sprintf("%s/bin/python main.py --server-name 0.0.0.0 --server-port 7860", venvPath)
    }

    // Jupyter notebooks
    if p.isJupyter(ctx) {
        if hasPoetry {
            return "poetry run jupyter lab --ip 0.0.0.0 --port 8888 --no-browser --allow-root"
        }
        venvPath := p.GetVenvPath(ctx)
        return fmt.Sprintf("%s/bin/jupyter lab --ip 0.0.0.0 --port 8888 --no-browser --allow-root", venvPath)
    }

    // Flask: prefer flask dev server if flask is present
    if p.isFlask(ctx) {
        if hasPoetry {
            if mainPythonFile != "" {
                return fmt.Sprintf("poetry run flask --app %s run --host 0.0.0.0 --port 5000", mainPythonFile)
            }
            return "poetry run flask run --host 0.0.0.0 --port 5000"
        }
        venvPath := p.GetVenvPath(ctx)
        if mainPythonFile != "" {
            return fmt.Sprintf("%s/bin/flask --app %s run --host 0.0.0.0 --port 5000", venvPath, mainPythonFile)
        }
        return fmt.Sprintf("%s/bin/flask run --host 0.0.0.0 --port 5000", venvPath)
    }

    if hasMainPythonFile {
        if hasPoetry {
            return fmt.Sprintf("poetry run python %s", mainPythonFile)
        }
        venvPath := p.GetVenvPath(ctx)
        return fmt.Sprintf("%s/bin/python %s", venvPath, mainPythonFile)
    }

    return ""
}

// GetDevStartCommandHost returns a development-friendly start command for host/local development
func (p *PythonProvider) GetDevStartCommandHost(ctx *generate.GenerateContext) string {
	// Check if this is a Poetry project
	hasPoetry := p.hasPoetry(ctx)
	
	// Django: use runserver with development setup
	if p.isDjango(ctx) {
		if hasPoetry {
			return "poetry run python manage.py runserver 0.0.0.0:8000"
		}
		venvPath := p.GetVenvPathForHost(ctx)
		return fmt.Sprintf("%s/bin/python manage.py runserver 0.0.0.0:8000", venvPath)
	}

    mainPythonFile := p.getMainPythonFile(ctx)
    hasMainPythonFile := mainPythonFile != ""

    // FastAPI (uvicorn) with reload if available
    if p.isFastAPI(ctx) && hasMainPythonFile {
        if hasPoetry {
            return "poetry run uvicorn main:app --reload --host 0.0.0.0 --port 8000"
        }
        venvPath := p.GetVenvPathForHost(ctx)
        return fmt.Sprintf("%s/bin/uvicorn main:app --reload --host 0.0.0.0 --port 8000", venvPath)
    }

    // Streamlit apps
    if p.isStreamlit(ctx) {
        if hasPoetry {
            return "poetry run streamlit run main.py --server.address 0.0.0.0 --server.port 8501"
        }
        venvPath := p.GetVenvPathForHost(ctx)
        return fmt.Sprintf("%s/bin/streamlit run main.py --server.address 0.0.0.0 --server.port 8501", venvPath)
    }

    // Gradio apps
    if p.isGradio(ctx) {
        if hasPoetry {
            return "poetry run python main.py --server-name 0.0.0.0 --server-port 7860"
        }
        venvPath := p.GetVenvPathForHost(ctx)
        return fmt.Sprintf("%s/bin/python main.py --server-name 0.0.0.0 --server-port 7860", venvPath)
    }

    // Jupyter notebooks
    if p.isJupyter(ctx) {
        if hasPoetry {
            return "poetry run jupyter lab --ip 0.0.0.0 --port 8888 --no-browser --allow-root"
        }
        venvPath := p.GetVenvPathForHost(ctx)
        return fmt.Sprintf("%s/bin/jupyter lab --ip 0.0.0.0 --port 8888 --no-browser --allow-root", venvPath)
    }

    // Flask: prefer flask dev server if flask is present
    if p.isFlask(ctx) {
        if hasPoetry {
            if mainPythonFile != "" {
                return fmt.Sprintf("poetry run flask --app %s run --host 0.0.0.0 --port 5000", mainPythonFile)
            }
            return "poetry run flask run --host 0.0.0.0 --port 5000"
        }
        venvPath := p.GetVenvPathForHost(ctx)
        if mainPythonFile != "" {
            return fmt.Sprintf("%s/bin/flask --app %s run --host 0.0.0.0 --port 5000", venvPath, mainPythonFile)
        }
        return fmt.Sprintf("%s/bin/flask run --host 0.0.0.0 --port 5000", venvPath)
    }

    if hasMainPythonFile {
        if hasPoetry {
            return fmt.Sprintf("poetry run python %s", mainPythonFile)
        }
        venvPath := p.GetVenvPathForHost(ctx)
        return fmt.Sprintf("%s/bin/python %s", venvPath, mainPythonFile)
    }

    return ""
}

// getDevPort returns the appropriate port for development mode based on framework
func (p *PythonProvider) getDevPort(ctx *generate.GenerateContext) string {
    if p.isDjango(ctx) {
        return "8000" // Django runserver default
    }
    if p.isFlask(ctx) {
        return "5000" // Flask default port
    }
    if p.isFastAPI(ctx) {
        return "8000" // FastAPI/uvicorn default
    }
    if p.isStreamlit(ctx) {
        return "8501" // Streamlit default port
    }
    if p.isGradio(ctx) {
        return "7860" // Gradio default port
    }
    if p.isJupyter(ctx) {
        return "8888" // Jupyter default port
    }
    if p.isFasthtml(ctx) {
        return "8000" // FastHTML default port
    }
    if p.isDataScience(ctx) {
        return "8888" // Data science apps often use Jupyter port
    }
    if p.isWebScraping(ctx) {
        return "8000" // Web scraping tools often use port 8000
    }
    return "8000" // Default fallback
}

func (p *PythonProvider) getMainPythonFile(ctx *generate.GenerateContext) string {
	// Framework-specific file detection
	if p.isStreamlit(ctx) {
		for _, file := range []string{"app.py", "main.py", "streamlit_app.py", "home.py"} {
			if ctx.App.HasMatch(file) {
				return file
			}
		}
	}
	
	if p.isGradio(ctx) {
		for _, file := range []string{"app.py", "main.py", "gradio_app.py", "interface.py"} {
			if ctx.App.HasMatch(file) {
				return file
			}
		}
	}
	
	if p.isJupyter(ctx) {
		for _, file := range []string{"notebook.ipynb", "main.ipynb", "app.ipynb", "analysis.ipynb"} {
			if ctx.App.HasMatch(file) {
				return file
			}
		}
	}
	
	if p.isDataScience(ctx) {
		for _, file := range []string{"analysis.py", "main.py", "app.py", "notebook.py", "data_analysis.py"} {
			if ctx.App.HasMatch(file) {
				return file
			}
		}
	}
	
	if p.isWebScraping(ctx) {
		for _, file := range []string{"scraper.py", "main.py", "app.py", "spider.py", "crawler.py"} {
			if ctx.App.HasMatch(file) {
				return file
			}
		}
	}
	
	// General Python file detection
	for _, file := range []string{"main.py", "app.py", "bot.py", "hello.py", "server.py", "index.py", "run.py", "start.py"} {
		if ctx.App.HasMatch(file) {
			return file
		}
	}
	return ""
}

func (p *PythonProvider) StartCommandHelp() string {
	return "To start your Python application, Railpack will automatically detect and configure:\n\n" +
		"**Web Frameworks:**\n" +
		"• Django projects with runserver (dev) or gunicorn (prod)\n" +
		"• Flask projects with flask run (dev) or gunicorn (prod)\n" +
		"• FastAPI projects with uvicorn\n" +
		"• FastHTML projects with uvicorn\n\n" +
		"**Data Science & ML:**\n" +
		"• Streamlit apps with streamlit run\n" +
		"• Gradio apps with python main.py\n" +
		"• Jupyter notebooks with jupyter lab\n" +
		"• Data science scripts with python\n\n" +
		"**Other Applications:**\n" +
		"• Web scraping tools with python\n" +
		"• General Python scripts with python\n\n" +
		"In development mode, your application will use development servers with hot reloading and appropriate ports for each framework."
}

func (p *PythonProvider) InstallUv(ctx *generate.GenerateContext, install *generate.CommandStepBuilder) []string {
	ctx.Logger.LogInfo("Using uv")

	venvPath := p.GetVenvPathForInstall(ctx)
	install.AddCache(ctx.Caches.AddCache("uv", UV_CACHE_DIR))
	install.AddEnvVars(map[string]string{
		"UV_COMPILE_BYTECODE": "1",
		"UV_LINK_MODE":        "copy",
		"UV_CACHE_DIR":        UV_CACHE_DIR,
		"UV_PYTHON_DOWNLOADS": "never",
		"VIRTUAL_ENV":         venvPath,
	})

	install.AddEnvVars(p.GetPythonEnvVars(ctx))

	p.copyInstallFiles(ctx, install)
	install.AddCommands([]plan.Command{
		plan.NewPathCommand(LOCAL_BIN_PATH),
		plan.NewPathCommand(venvPath + "/bin"),
		// Combined command: create venv and sync dependencies
		// if we exclude workspace packages, uv.lock will fail the frozen test and the user will get an error
		// to avoid this, we (a) detect if workspace packages are required (b) if they aren't, we don't include project
		// source in order to optimize layer caching (c) install project in the build phase.
		plan.NewExecCommand(fmt.Sprintf("python -m venv %s && %s/bin/pip install uv && %s/bin/uv sync --locked --no-dev --no-install-project", venvPath, venvPath, venvPath)),
	})

	return []string{venvPath}
}

func (p *PythonProvider) InstallPipenv(ctx *generate.GenerateContext, install *generate.CommandStepBuilder) []string {
	ctx.Logger.LogInfo("Using pipenv")

	venvPath := p.GetVenvPathForInstall(ctx)
	install.AddEnvVars(p.GetPythonEnvVars(ctx))
	install.AddEnvVars(map[string]string{
		"PIPENV_CHECK_UPDATE":       "false",
		"PIPENV_VENV_IN_PROJECT":    "1",
		"PIPENV_IGNORE_VIRTUALENVS": "1",
	})

	install.AddCommands([]plan.Command{
		plan.NewPathCommand(LOCAL_BIN_PATH),
		plan.NewPathCommand(venvPath + "/bin"),
	})

	if ctx.App.HasMatch("Pipfile.lock") {
		install.AddCommands([]plan.Command{
			plan.NewCopyCommand("Pipfile"),
			plan.NewCopyCommand("Pipfile.lock"),
			// Combined command: create venv and install with pipenv
			plan.NewExecCommand(fmt.Sprintf("python -m venv %s && %s/bin/pip install pipenv && %s/bin/pipenv install --deploy --ignore-pipfile", venvPath, venvPath, venvPath)),
		})
	} else {
		install.AddCommands([]plan.Command{
			plan.NewCopyCommand("Pipfile"),
			// Combined command: create venv and install with pipenv
			plan.NewExecCommand(fmt.Sprintf("python -m venv %s && %s/bin/pip install pipenv && %s/bin/pipenv install --skip-lock", venvPath, venvPath, venvPath)),
		})
	}

	return []string{venvPath}
}

func (p *PythonProvider) InstallPDM(ctx *generate.GenerateContext, install *generate.CommandStepBuilder) []string {
	ctx.Logger.LogInfo("Using pdm")

	venvPath := p.GetVenvPathForInstall(ctx)
	install.AddEnvVars(p.GetPythonEnvVars(ctx))
	install.AddEnvVars(map[string]string{
		"PDM_CHECK_UPDATE": "false",
	})

	p.copyInstallFiles(ctx, install)
	install.AddCommands([]plan.Command{
		plan.NewPathCommand(LOCAL_BIN_PATH),
		plan.NewPathCommand(venvPath + "/bin"),
		// Combined command: create venv and install with pdm
		plan.NewExecCommand(fmt.Sprintf("python -m venv %s && %s/bin/pip install pdm && %s/bin/pdm install --check --prod --no-editable", venvPath, venvPath, venvPath)),
	})

	return []string{venvPath}
}

func (p *PythonProvider) InstallPoetry(ctx *generate.GenerateContext, install *generate.CommandStepBuilder) []string {
	ctx.Logger.LogInfo("Using poetry")

	venvPath := p.GetVenvPathForInstall(ctx)
	install.AddEnvVars(p.GetPythonEnvVars(ctx))
	install.AddEnvVars(map[string]string{
		"VIRTUAL_ENV":                   venvPath,
		"POETRY_VIRTUALENVS_PATH":       venvPath,
		"POETRY_VIRTUALENVS_IN_PROJECT": "true",
	})

	p.copyInstallFiles(ctx, install)
	install.AddCommands([]plan.Command{
		plan.NewPathCommand(LOCAL_BIN_PATH),
		plan.NewPathCommand(venvPath + "/bin"),
		// Combined command: create venv and install with poetry
		plan.NewExecCommand(fmt.Sprintf("python -m venv %s && %s/bin/pip install poetry && %s/bin/poetry install --no-interaction --no-ansi --only main --no-root", venvPath, venvPath, venvPath)),
	})

	return []string{venvPath}
}

func (p *PythonProvider) InstallPip(ctx *generate.GenerateContext, install *generate.CommandStepBuilder) []string {
	ctx.Logger.LogInfo("Using pip")

	venvPath := p.GetVenvPathForInstall(ctx)
	install.AddCache(ctx.Caches.AddCache("pip", PIP_CACHE_DIR))
	install.AddEnvVars(p.GetPythonEnvVars(ctx))
	install.AddEnvVars(map[string]string{
		"PIP_CACHE_DIR": PIP_CACHE_DIR,
		"VIRTUAL_ENV":   venvPath,
	})

	// Copy requirements.txt before installing
	p.copyInstallFiles(ctx, install)
	
	// Combined command: create venv and install dependencies
	install.AddCommands([]plan.Command{
		plan.NewExecCommand(fmt.Sprintf("python -m venv %s && %s/bin/pip install -r requirements.txt", venvPath, venvPath)),
		plan.NewPathCommand(venvPath + "/bin"),
	})

	return []string{venvPath}
}

func (p *PythonProvider) AddRuntimeDeps(ctx *generate.GenerateContext) {
	for dep, requiredPkgs := range pythonRuntimeDepRequirements {
		if p.usesDep(ctx, dep) {
			ctx.Logger.LogInfo("Installing runtime apt packages for %s", dep)
			ctx.Deploy.AddAptPackages(requiredPkgs)
		}
	}

	if p.usesPostgres(ctx) {
		ctx.Deploy.AddAptPackages([]string{"libpq5"})
	}

	if p.usesMysql(ctx) {
		ctx.Deploy.AddAptPackages([]string{"default-mysql-client"})
	}
}

func (p *PythonProvider) GetBuilderDeps(ctx *generate.GenerateContext) *generate.MiseStepBuilder {
	miseStep := ctx.GetMiseStepBuilder()
	miseStep.SupportingAptPackages = append(miseStep.SupportingAptPackages, "python3-dev", "gcc", "g++", "libc6-dev", "build-essential")

	for dep, requiredPkgs := range pythonBuildDepRequirements {
		if p.usesDep(ctx, dep) {
			ctx.Logger.LogInfo("Installing build apt packages for %s", dep)
			miseStep.SupportingAptPackages = append(miseStep.SupportingAptPackages, requiredPkgs...)
		}
	}

	if p.usesPostgres(ctx) {
		miseStep.SupportingAptPackages = append(miseStep.SupportingAptPackages, "libpq-dev")
	}

	if p.usesMysql(ctx) {
		miseStep.SupportingAptPackages = append(miseStep.SupportingAptPackages, "default-libmysqlclient-dev")
	}

	return miseStep
}

func (p *PythonProvider) InstallMisePackages(ctx *generate.GenerateContext, miseStep *generate.MiseStepBuilder) {
	python := miseStep.Default("python", DEFAULT_PYTHON_VERSION)

	if envVersion, varName := ctx.Env.GetConfigVariable("PYTHON_VERSION"); envVersion != "" {
		miseStep.Version(python, envVersion, varName)
	}

	if versionFile, err := ctx.App.ReadFile(".python-version"); err == nil {
		miseStep.Version(python, utils.ExtractSemverVersion(string(versionFile)), ".python-version")
	}

	if runtimeFile, err := ctx.App.ReadFile("runtime.txt"); err == nil {
		miseStep.Version(python, utils.ExtractSemverVersion(string(runtimeFile)), "runtime.txt")
	}

	if pipfileVersion, pipfileVarName := parseVersionFromPipfile(ctx); pipfileVersion != "" {
		miseStep.Version(python, pipfileVersion, fmt.Sprintf("Pipfile > %s", pipfileVarName))
	}

	if p.hasPoetry(ctx) || p.hasUv(ctx) || p.hasPdm(ctx) || p.hasPipfile(ctx) {
		miseStep.Default("pipx", "latest")
	}

	if p.hasPoetry(ctx) {
		miseStep.Default("pipx:poetry", "latest")
	}

	if p.hasPdm(ctx) {
		miseStep.Default("pipx:pdm", "latest")
	}

	if p.hasUv(ctx) {
		miseStep.Default("pipx:uv", "latest")
	}

	if p.hasPipfile(ctx) {
		miseStep.Default("pipx:pipenv", "latest")
	}

}

func (p *PythonProvider) GetPythonEnvVars(ctx *generate.GenerateContext) map[string]string {
	return map[string]string{
		"PYTHONFAULTHANDLER":            "1",
		"PYTHONUNBUFFERED":              "1",
		"PYTHONHASHSEED":                "random",
		"PYTHONDONTWRITEBYTECODE":       "1",
		"PIP_DISABLE_PIP_VERSION_CHECK": "1",
		"PIP_DEFAULT_TIMEOUT":           "100",
	}
}

func (p *PythonProvider) GetPythonDevEnvVars(ctx *generate.GenerateContext) map[string]string {
	envVars := map[string]string{
		"PYTHONFAULTHANDLER":            "1",
		"PYTHONUNBUFFERED":              "1",
		"PYTHONHASHSEED":                "random",
		"PYTHONDONTWRITEBYTECODE":       "1",
		"PIP_DISABLE_PIP_VERSION_CHECK": "1",
		"PIP_DEFAULT_TIMEOUT":           "100",
	}

	// Add development-specific environment variables
	if ctx.Dev {
		// Framework-specific development settings
		if p.isDjango(ctx) {
			// Django uses databases by default
			envVars["DATABASE_URL"] = "sqlite:///db.sqlite3"
			envVars["DJANGO_SETTINGS_MODULE"] = "mysite.settings"
			envVars["DJANGO_COLLECTSTATIC"] = "False"
			envVars["DJANGO_DEBUG"] = "True"
		} else if p.isFlask(ctx) {
			// Flask might use databases, check if it has database dependencies
			if p.isDatabase(ctx) {
				envVars["DATABASE_URL"] = "sqlite:///db.sqlite3"
			}
			envVars["FLASK_APP"] = "main.py"
			envVars["FLASK_ENV"] = "development"
			envVars["FLASK_DEBUG"] = "True"
		} else if p.isFastAPI(ctx) {
			// FastAPI might use databases, check if it has database dependencies
			if p.isDatabase(ctx) {
				envVars["DATABASE_URL"] = "sqlite:///db.sqlite3"
			}
			envVars["FASTAPI_ENV"] = "development"
			envVars["FASTAPI_DEBUG"] = "True"
		} else if p.isStreamlit(ctx) {
			envVars["STREAMLIT_SERVER_HEADLESS"] = "true"
			envVars["STREAMLIT_SERVER_ENABLE_CORS"] = "false"
			envVars["STREAMLIT_SERVER_PORT"] = "8501"
		} else if p.isGradio(ctx) {
			envVars["GRADIO_SERVER_NAME"] = "0.0.0.0"
			envVars["GRADIO_SERVER_PORT"] = "7860"
		} else if p.isJupyter(ctx) {
			envVars["JUPYTER_ENABLE_LAB"] = "yes"
			envVars["JUPYTER_ALLOW_ROOT"] = "yes"
		} else if p.isDataScience(ctx) {
			envVars["MPLBACKEND"] = "Agg" // Use non-interactive backend for matplotlib
		} else if p.isWebScraping(ctx) {
			envVars["DISPLAY"] = ":99" // For headless browser testing
		}
		
		// Only add container-specific variables for production
		// In development, these are not needed
	}

	return envVars
}

// GetPythonProdEnvVars returns production-specific environment variables
func (p *PythonProvider) GetPythonProdEnvVars(ctx *generate.GenerateContext) map[string]string {
	envVars := p.GetPythonDevEnvVars(ctx)
	
	// Add production-specific variables
	envVars["IN_CONTAINER"] = "1"
	envVars["PYTHONPATH"] = "/app"
	
	// Add MPLBACKEND for data science apps in production
	if p.isDataScience(ctx) {
		envVars["MPLBACKEND"] = "Agg"
	}
	
	return envVars
}

func (p *PythonProvider) copyInstallFiles(ctx *generate.GenerateContext, install *generate.CommandStepBuilder) {
	if p.installNeedsAllFiles(ctx) {
		install.AddInput(plan.NewLocalLayer())
		return
	}

	patterns := []string{
		"requirements.txt",
		"pyproject.toml",
		"Pipfile",
		"poetry.lock",
		"uv.lock",
		"pdm.lock",
	}

	for _, pattern := range patterns {
		if files, err := ctx.App.FindFiles(pattern); err == nil {
			for _, file := range files {
				install.AddCommand(plan.NewCopyCommand(file))
			}
		}
	}
}

// inspect python dependency files and determine if local packages are referenced, and therefore all files are required
// for installation.
func (p *PythonProvider) installNeedsAllFiles(ctx *generate.GenerateContext) bool {
	if requirementsContent, err := ctx.App.ReadFile("requirements.txt"); err == nil {
		return strings.Contains(requirementsContent, "file://")
	}

	// inspect pyproject.toml for local path references or uv workspace usage
	if pyprojectContent, err := ctx.App.ReadFile("pyproject.toml"); err == nil {
		if strings.Contains(pyprojectContent, "file://") || strings.Contains(pyprojectContent, "path = ") {
			return true
		}
	}

	// TODO just having a `uv.tool.workspace` key doesn't necessarily mean you are listing a workspace item as a dependency
	// parse TOML using existing helper to check for tool.uv.workspace key
	var pyproject map[string]any
	if err := ctx.App.ReadTOML("pyproject.toml", &pyproject); err == nil {
		if tool, ok := pyproject["tool"].(map[string]any); ok {
			if uv, ok := tool["uv"].(map[string]any); ok {
				if _, exists := uv["workspace"]; exists {
					return true
				}
			}
		}
	} else {
		log.Infof("Failed to read pyproject.toml: %v", err)
	}

	return false
}

func (p *PythonProvider) usesPostgres(ctx *generate.GenerateContext) bool {
	djangoPythonRe := regexp.MustCompile(`django.db.backends.postgresql`)
	containsDjangoPostgres := len(ctx.App.FindFilesWithContent("**/*.py", djangoPythonRe)) > 0
	return p.usesDep(ctx, "psycopg2") || p.usesDep(ctx, "psycopg2-binary") || containsDjangoPostgres
}

func (p *PythonProvider) usesMysql(ctx *generate.GenerateContext) bool {
	djangoPythonRe := regexp.MustCompile(`django.db.backends.mysql`)
	containsDjangoMysql := len(ctx.App.FindFilesWithContent("**/*.py", djangoPythonRe)) > 0
	return p.usesDep(ctx, "mysqlclient") || containsDjangoMysql
}

func (p *PythonProvider) addMetadata(ctx *generate.GenerateContext) {
	hasPoetry := p.hasPoetry(ctx)
	hasPdm := p.hasPdm(ctx)
	hasUv := p.hasUv(ctx)

	pkgManager := "pip"

	if hasPoetry {
		pkgManager = "poetry"
	} else if hasPdm {
		pkgManager = "pdm"
	} else if hasUv {
		pkgManager = "uv"
	}

	ctx.Metadata.Set("pythonPackageManager", pkgManager)
	ctx.Metadata.Set("pythonRuntime", p.getRuntime(ctx))
}

func (p *PythonProvider) usesDep(ctx *generate.GenerateContext, dep string) bool {
	for _, file := range []string{"requirements.txt", "pyproject.toml", "Pipfile"} {
		if contents, err := ctx.App.ReadFile(file); err == nil {
			// TODO: Do something better than string comparison
			if strings.Contains(strings.ToLower(contents), strings.ToLower(dep)) {
				return true
			}
		}
	}
	return false
}

var pipfileFullVersionRegex = regexp.MustCompile(`python_full_version\s*=\s*['"]([0-9.]*)"?`)
var pipfileShortVersionRegex = regexp.MustCompile(`python_version\s*=\s*['"]([0-9.]*)"?`)

func parseVersionFromPipfile(ctx *generate.GenerateContext) (string, string) {
	pipfile, err := ctx.App.ReadFile("Pipfile")
	if err != nil {
		return "", ""
	}

	if matches := pipfileFullVersionRegex.FindStringSubmatch(string(pipfile)); len(matches) > 1 {
		return matches[1], "python_full_version"
	}

	if matches := pipfileShortVersionRegex.FindStringSubmatch(string(pipfile)); len(matches) > 1 {
		return matches[1], "python_version"
	}

	return "", ""
}

func (p *PythonProvider) hasRequirements(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("requirements.txt")
}

func (p *PythonProvider) hasPyproject(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("pyproject.toml")
}

func (p *PythonProvider) hasPipfile(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("Pipfile")
}

func (p *PythonProvider) hasPoetry(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("poetry.lock")
}

func (p *PythonProvider) hasPdm(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("pdm.lock")
}

func (p *PythonProvider) hasUv(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("uv.lock")
}

func (p *PythonProvider) isFasthtml(ctx *generate.GenerateContext) bool {
	return p.usesDep(ctx, "python-fasthtml")
}

func (p *PythonProvider) isFlask(ctx *generate.GenerateContext) bool {
	return p.usesDep(ctx, "flask")
}

func (p *PythonProvider) isFastAPI(ctx *generate.GenerateContext) bool {
	return p.usesDep(ctx, "fastapi") || p.usesDep(ctx, "uvicorn")
}

func (p *PythonProvider) isStreamlit(ctx *generate.GenerateContext) bool {
	return p.usesDep(ctx, "streamlit")
}

func (p *PythonProvider) isGradio(ctx *generate.GenerateContext) bool {
	return p.usesDep(ctx, "gradio")
}

func (p *PythonProvider) isJupyter(ctx *generate.GenerateContext) bool {
	return p.usesDep(ctx, "jupyter") || p.usesDep(ctx, "notebook") || p.usesDep(ctx, "jupyterlab")
}

func (p *PythonProvider) isDataScience(ctx *generate.GenerateContext) bool {
	return p.usesDep(ctx, "pandas") || p.usesDep(ctx, "numpy") || p.usesDep(ctx, "matplotlib") || 
		   p.usesDep(ctx, "seaborn") || p.usesDep(ctx, "scikit-learn") || p.usesDep(ctx, "tensorflow") || 
		   p.usesDep(ctx, "pytorch") || p.usesDep(ctx, "keras")
}

func (p *PythonProvider) isWebScraping(ctx *generate.GenerateContext) bool {
	return p.usesDep(ctx, "requests") || p.usesDep(ctx, "scrapy") || p.usesDep(ctx, "beautifulsoup4") || 
		   p.usesDep(ctx, "selenium") || p.usesDep(ctx, "playwright")
}

func (p *PythonProvider) isDatabase(ctx *generate.GenerateContext) bool {
	return p.usesDep(ctx, "sqlalchemy") || p.usesDep(ctx, "psycopg2") || p.usesDep(ctx, "mysqlclient") || 
		   p.usesDep(ctx, "pymongo") || p.usesDep(ctx, "redis") || p.usesDep(ctx, "databases") ||
		   p.usesDep(ctx, "flask-sqlalchemy") || p.usesDep(ctx, "django") || p.usesDep(ctx, "tortoise-orm")
}

func (p *PythonProvider) getRuntime(ctx *generate.GenerateContext) string {
	if p.isDjango(ctx) {
		return "django"
	} else if p.isFlask(ctx) {
		return "flask"
	} else if p.isFastAPI(ctx) {
		return "fastapi"
	} else if p.isFasthtml(ctx) {
		return "fasthtml"
	} else if p.isStreamlit(ctx) {
		return "streamlit"
	} else if p.isGradio(ctx) {
		return "gradio"
	} else if p.isJupyter(ctx) {
		return "jupyter"
	} else if p.isDataScience(ctx) {
		return "data-science"
	} else if p.isWebScraping(ctx) {
		return "web-scraping"
	}

	return "python"
}

// Mapping of python dependencies to required apt packages

var pythonBuildDepRequirements = map[string][]string{
	"pycairo":     {"libcairo2-dev"},
	"pillow":      {"libjpeg-dev", "zlib1g-dev", "libpng-dev"},
	"opencv":      {"libopencv-dev", "libglib2.0-0", "libsm6", "libxext6", "libxrender-dev", "libgomp1"},
	"numpy":       {"libopenblas-dev", "liblapack-dev", "gfortran"},
	"scipy":       {"libopenblas-dev", "liblapack-dev", "gfortran"},
	"pandas":      {"libopenblas-dev", "liblapack-dev", "gfortran"},
	"matplotlib":  {"libfreetype6-dev", "libpng-dev"},
	"scikit-learn": {"libopenblas-dev", "liblapack-dev", "gfortran"},
	"tensorflow":  {"libcudnn8", "libcudnn8-dev", "libcublas11", "libcublas-dev"},
	"pytorch":     {"libcudnn8", "libcudnn8-dev", "libcublas11", "libcublas-dev"},
	"lxml":        {"libxml2-dev", "libxslt1-dev"},
	"cryptography": {"libssl-dev", "libffi-dev"},
	"psycopg2":    {"libpq-dev"},
	"mysqlclient": {"default-libmysqlclient-dev"},
	"redis":       {"redis-server"},
	"selenium":    {"chromium-browser", "chromium-chromedriver"},
	"playwright":  {"chromium-browser", "chromium-chromedriver"},
}

var pythonRuntimeDepRequirements = map[string][]string{
	"pycairo":     {"libcairo2"},
	"pdf2image":   {"poppler-utils"},
	"pydub":       {"ffmpeg"},
	"pymovie":     {"ffmpeg", "qt5-qmake", "qtbase5-dev", "qtbase5-dev-tools", "qttools5-dev-tools", "libqt5core5a", "python3-pyqt5"},
	"pillow":      {"libjpeg62-turbo", "zlib1g", "libpng16-16"},
	"opencv":      {"libopencv-core4.5", "libopencv-imgproc4.5", "libopencv-imgcodecs4.5", "libglib2.0-0", "libsm6", "libxext6", "libxrender1", "libgomp1"},
	"numpy":       {"libopenblas0", "liblapack3", "gfortran"},
	"scipy":       {"libopenblas0", "liblapack3", "gfortran"},
	"pandas":      {"libopenblas0", "liblapack3", "gfortran"},
	"matplotlib":  {"libfreetype6", "libpng16-16"},
	"scikit-learn": {"libopenblas0", "liblapack3", "gfortran"},
	"tensorflow":  {"libcudnn8", "libcublas11"},
	"pytorch":     {"libcudnn8", "libcublas11"},
	"lxml":        {"libxml2", "libxslt1.1"},
	"cryptography": {"libssl1.1", "libffi7"},
	"psycopg2":    {"libpq5"},
	"mysqlclient": {"default-mysql-client"},
	"redis":       {"redis-server"},
	"selenium":    {"chromium-browser", "chromium-chromedriver"},
	"playwright":  {"chromium-browser", "chromium-chromedriver"},
}
