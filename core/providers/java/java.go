package java

import (
	"fmt"
	"strings"

	"github.com/railwayapp/railpack/core/generate"
	"github.com/railwayapp/railpack/core/plan"
)

type JavaProvider struct{}

func (p *JavaProvider) Name() string {
	return "java"
}

func (p *JavaProvider) Detect(ctx *generate.GenerateContext) (bool, error) {
	return ctx.App.HasMatch("pom.{xml,atom,clj,groovy,rb,scala,yaml,yml}") || ctx.App.HasMatch("gradlew"), nil
}

func (p *JavaProvider) Initialize(ctx *generate.GenerateContext) error {
	return nil
}

func (p *JavaProvider) StartCommandHelp() string {
	return ""
}

func (p *JavaProvider) Plan(ctx *generate.GenerateContext) error {
	build := ctx.NewCommandStep("build")
	build.AddInput(plan.NewStepLayer(ctx.GetMiseStepBuilder().Name()))
	build.AddInput(plan.NewLocalLayer())

	if p.usesGradle(ctx) {
		ctx.Logger.LogInfo("Using Gradle")

		p.setGradleVersion(ctx)
		p.setJDKVersion(ctx, ctx.GetMiseStepBuilder())

		if ctx.App.HasMatch("gradlew") && !ctx.App.IsFileExecutable("gradlew") {
			build.AddCommand(plan.NewExecCommand("chmod +x gradlew"))
		}

		build.AddCommand(plan.NewExecCommand("./gradlew clean build -x check -x test -Pproduction"))
		build.AddCache(p.gradleCache(ctx))
	} else {
		ctx.Logger.LogInfo("Using Maven")

		ctx.GetMiseStepBuilder().Default("maven", "latest")
		p.setJDKVersion(ctx, ctx.GetMiseStepBuilder())

		if ctx.App.HasMatch("mvnw") && !ctx.App.IsFileExecutable("mvnw") {
			build.AddCommand(plan.NewExecCommand("chmod +x mvnw"))
		}

		build.AddCommand(plan.NewExecCommand(fmt.Sprintf("%s -DoutputFile=target/mvn-dependency-list.log -B -DskipTests clean dependency:list install -Pproduction", p.getMavenExe(ctx))))
		build.AddCache(p.mavenCache(ctx))
	}

	runtimeMiseStep := ctx.NewMiseStepBuilder("packages:mise:runtime")
	p.setJDKVersion(ctx, runtimeMiseStep)

	outPath := "target/."
	if ctx.App.HasMatch("**/build/libs/*.jar") || p.usesGradle(ctx) {
		outPath = "."
	}

	ctx.Deploy.AddInputs([]plan.Layer{
		runtimeMiseStep.GetLayer(),
		plan.NewStepLayer(build.Name(), plan.Filter{
			Include: []string{outPath},
		}),
	})

	ctx.Deploy.StartCmd = p.getStartCmd(ctx)

	if ctx.Dev {
		if dev := p.getDevStartCmd(ctx); dev != "" {
			ctx.Deploy.StartCmd = dev
		}
		// Add host-specific development command
		if devHost := p.getDevStartCmdHost(ctx); devHost != "" {
			ctx.Deploy.StartCmdHost = devHost
		}
		// Add development environment variables
		ctx.Deploy.Variables = p.getJavaDevEnvVars(ctx)
		// Add required port for web applications
		if port := p.getDevPort(ctx); port != "" {
			ctx.Deploy.RequiredPort = port
		}
	} else {
		// Add production environment variables
		ctx.Deploy.Variables = p.getJavaProdEnvVars(ctx)
	}

	p.addMetadata(ctx)

	return nil
}

func (p *JavaProvider) getStartCmd(ctx *generate.GenerateContext) string {
	if p.usesGradle(ctx) {
		buildGradle := p.readBuildGradle(ctx)
		return fmt.Sprintf("java $JAVA_OPTS -jar %s $(ls -1 */build/libs/*jar | grep -v plain)", getGradlePortConfig(buildGradle))
	} else if ctx.App.HasMatch("pom.xml") {
		return fmt.Sprintf("java %s $JAVA_OPTS -jar target/*jar", getMavenPortConfig(ctx))
	} else {
		return "java $JAVA_OPTS -jar target/*jar"
	}

}

func (p *JavaProvider) getDevStartCmd(ctx *generate.GenerateContext) string {
	if p.usesGradle(ctx) {
		if ctx.App.HasMatch("gradlew") {
			return "./gradlew run"
		}
		return "gradle run"
	}
	if ctx.App.HasMatch("pom.xml") {
		if p.usesSpringBoot(ctx) {
			return "mvn spring-boot:run"
		}
		// Fallback: leave empty to keep prod jar run
		return ""
	}
	return ""
}

func (p *JavaProvider) getDevStartCmdHost(ctx *generate.GenerateContext) string {
	if p.usesGradle(ctx) {
		if ctx.App.HasMatch("gradlew") {
			return "./gradlew run"
		}
		return "gradle run"
	}
	if ctx.App.HasMatch("pom.xml") {
		if p.usesSpringBoot(ctx) {
			return "mvn spring-boot:run"
		}
		// Fallback: leave empty to keep prod jar run
		return ""
	}
	return ""
}

func (p *JavaProvider) addMetadata(ctx *generate.GenerateContext) {
	hasGradle := p.usesGradle(ctx)

	if hasGradle {
		ctx.Metadata.Set("javaPackageManager", "gradle")
	} else {
		ctx.Metadata.Set("javaPackageManager", "maven")
	}

	var framework string
	if p.usesSpringBoot(ctx) {
		framework = "spring-boot"
	}

	ctx.Metadata.Set("javaFramework", framework)
}

func (p *JavaProvider) usesSpringBoot(ctx *generate.GenerateContext) bool {
	return ctx.App.HasMatch("**/spring-boot*.jar") ||
		ctx.App.HasMatch("**/spring-boot*.class") ||
		ctx.App.HasMatch("**/org/springframework/boot/**")
}

func (p *JavaProvider) getJavaDevEnvVars(ctx *generate.GenerateContext) map[string]string {
	envVars := make(map[string]string)

	// Development-specific environment variables
	envVars["JAVA_OPTS"] = "-Xmx512m -Xms256m"
	envVars["SPRING_PROFILES_ACTIVE"] = "development"

	// Framework-specific development settings
	if p.usesSpringBoot(ctx) {
		envVars["SPRING_DEVTOOLS_RESTART_ENABLED"] = "true"
		envVars["SPRING_DEVTOOLS_LIVERELOAD_ENABLED"] = "true"
		envVars["SPRING_DEVTOOLS_ADD_PROPERTIES"] = "true"
		envVars["SPRING_JPA_SHOW_SQL"] = "true"
		envVars["SPRING_JPA_PROPERTIES_HIBERNATE_FORMAT_SQL"] = "true"
	}

	// Gradle-specific development settings
	if p.usesGradle(ctx) {
		envVars["GRADLE_OPTS"] = "-Xmx512m -Dfile.encoding=UTF-8"
		envVars["GRADLE_USER_HOME"] = "/root/.gradle"
	}

	// Maven-specific development settings
	if ctx.App.HasMatch("pom.xml") {
		envVars["MAVEN_OPTS"] = "-Xmx512m -Dfile.encoding=UTF-8"
		envVars["MAVEN_CONFIG"] = "/root/.m2"
	}

	return envVars
}

func (p *JavaProvider) getJavaProdEnvVars(ctx *generate.GenerateContext) map[string]string {
	envVars := make(map[string]string)

	// Production-specific environment variables
	envVars["JAVA_OPTS"] = "-Xmx1024m -Xms512m -XX:+UseG1GC"
	envVars["SPRING_PROFILES_ACTIVE"] = "production"

	// Framework-specific production settings
	if p.usesSpringBoot(ctx) {
		envVars["SPRING_JPA_SHOW_SQL"] = "false"
		envVars["SPRING_JPA_PROPERTIES_HIBERNATE_FORMAT_SQL"] = "false"
		envVars["SPRING_DEVTOOLS_RESTART_ENABLED"] = "false"
		envVars["SPRING_DEVTOOLS_LIVERELOAD_ENABLED"] = "false"
	}

	// Gradle-specific production settings
	if p.usesGradle(ctx) {
		envVars["GRADLE_OPTS"] = "-Xmx1024m -Dfile.encoding=UTF-8"
		envVars["GRADLE_USER_HOME"] = "/root/.gradle"
	}

	// Maven-specific production settings
	if ctx.App.HasMatch("pom.xml") {
		envVars["MAVEN_OPTS"] = "-Xmx1024m -Dfile.encoding=UTF-8"
		envVars["MAVEN_CONFIG"] = "/root/.m2"
	}

	return envVars
}

// getDevPort returns the appropriate port for development mode based on framework
func (p *JavaProvider) getDevPort(ctx *generate.GenerateContext) string {
	// Spring Boot applications
	if p.usesSpringBoot(ctx) {
		return "8080" // Spring Boot default port
	}

	// Check for common Java web frameworks
	if p.usesGradle(ctx) {
		buildGradle := p.readBuildGradle(ctx)
		if strings.Contains(buildGradle, "spring-boot") {
			return "8080" // Spring Boot default
		}
		if strings.Contains(buildGradle, "quarkus") {
			return "8080" // Quarkus default
		}
		if strings.Contains(buildGradle, "micronaut") {
			return "8080" // Micronaut default
		}
		if strings.Contains(buildGradle, "vertx") {
			return "8080" // Vert.x default
		}
		if strings.Contains(buildGradle, "play") {
			return "9000" // Play Framework default
		}
	}

	if ctx.App.HasMatch("pom.xml") {
		pomFile, err := ctx.App.ReadFile("pom.xml")
		if err == nil {
			if strings.Contains(pomFile, "spring-boot") {
				return "8080" // Spring Boot default
			}
			if strings.Contains(pomFile, "quarkus") {
				return "8080" // Quarkus default
			}
			if strings.Contains(pomFile, "micronaut") {
				return "8080" // Micronaut default
			}
			if strings.Contains(pomFile, "vertx") {
				return "8080" // Vert.x default
			}
			if strings.Contains(pomFile, "play") {
				return "9000" // Play Framework default
			}
		}
	}

	// Check for web server dependencies
	if p.usesWebServer(ctx) {
		return "8080" // Default Java web server port
	}

	// Default port for all Java applications (including console apps)
	return "8080"
}

// usesWebServer checks if the application uses a web server
func (p *JavaProvider) usesWebServer(ctx *generate.GenerateContext) bool {
	// Check for common web server dependencies
	webDeps := []string{
		"spring-boot-starter-web",
		"spring-boot-starter-tomcat",
		"spring-boot-starter-jetty",
		"spring-boot-starter-undertow",
		"quarkus-resteasy",
		"quarkus-vertx-web",
		"micronaut-http-server",
		"vertx-web",
		"play-framework",
		"javax.servlet",
		"jakarta.servlet",
	}

	// Check Gradle build file
	if p.usesGradle(ctx) {
		buildGradle := p.readBuildGradle(ctx)
		for _, dep := range webDeps {
			if strings.Contains(buildGradle, dep) {
				return true
			}
		}
	}

	// Check Maven pom.xml
	if ctx.App.HasMatch("pom.xml") {
		pomFile, err := ctx.App.ReadFile("pom.xml")
		if err == nil {
			for _, dep := range webDeps {
				if strings.Contains(pomFile, dep) {
					return true
				}
			}
		}
	}

	return false
}
