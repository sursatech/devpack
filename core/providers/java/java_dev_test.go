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

func TestJava_Dev_Gradle_HasStartCmdHost(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/java-gradle")
	ctx.Dev = true

	provider := JavaProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	// Should have StartCmdHost for development mode
	require.NotEmpty(t, ctx.Deploy.StartCmdHost)
	require.Contains(t, ctx.Deploy.StartCmdHost, "gradle")
	require.Contains(t, ctx.Deploy.StartCmdHost, "run")
}

func TestJava_Dev_Maven_HasStartCmdHost(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/java-maven")
	ctx.Dev = true

	provider := JavaProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	// Maven example is not Spring Boot, so StartCmdHost should be empty
	// (falls back to production jar run)
	require.Empty(t, ctx.Deploy.StartCmdHost)
}

func TestJava_Dev_HasDevEnvVars(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/java-gradle")
	ctx.Dev = true

	provider := JavaProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	// Should have development environment variables
	require.NotNil(t, ctx.Deploy.Variables)
	require.Equal(t, "development", ctx.Deploy.Variables["SPRING_PROFILES_ACTIVE"])
	require.Equal(t, "-Xmx512m -Xms256m", ctx.Deploy.Variables["JAVA_OPTS"])
	require.Equal(t, "-Xmx512m -Dfile.encoding=UTF-8", ctx.Deploy.Variables["GRADLE_OPTS"])
}

func TestJava_Prod_HasProdEnvVars(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/java-gradle")
	ctx.Dev = false

	provider := JavaProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	// Should have production environment variables
	require.NotNil(t, ctx.Deploy.Variables)
	require.Equal(t, "production", ctx.Deploy.Variables["SPRING_PROFILES_ACTIVE"])
	require.Equal(t, "-Xmx1024m -Xms512m -XX:+UseG1GC", ctx.Deploy.Variables["JAVA_OPTS"])
	require.Equal(t, "-Xmx1024m -Dfile.encoding=UTF-8", ctx.Deploy.Variables["GRADLE_OPTS"])
}

func TestJava_Dev_SpringBoot_HasSpringDevVars(t *testing.T) {
	// Test with a non-Spring Boot project - should not have Spring Boot variables
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/java-maven")
	ctx.Dev = true

	provider := JavaProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	// Should have basic development variables but not Spring Boot specific ones
	require.NotNil(t, ctx.Deploy.Variables)
	require.Equal(t, "development", ctx.Deploy.Variables["SPRING_PROFILES_ACTIVE"])
	require.Equal(t, "-Xmx512m -Xms256m", ctx.Deploy.Variables["JAVA_OPTS"])

	// Should not have Spring Boot specific variables since it's not a Spring Boot project
	require.Empty(t, ctx.Deploy.Variables["SPRING_DEVTOOLS_RESTART_ENABLED"])
	require.Empty(t, ctx.Deploy.Variables["SPRING_DEVTOOLS_LIVERELOAD_ENABLED"])
	require.Empty(t, ctx.Deploy.Variables["SPRING_JPA_SHOW_SQL"])
}

func TestJava_Prod_SpringBoot_HasSpringProdVars(t *testing.T) {
	// Test with a non-Spring Boot project - should not have Spring Boot variables
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/java-maven")
	ctx.Dev = false

	provider := JavaProvider{}
	err := provider.Plan(ctx)
	require.NoError(t, err)

	// Should have basic production variables but not Spring Boot specific ones
	require.NotNil(t, ctx.Deploy.Variables)
	require.Equal(t, "production", ctx.Deploy.Variables["SPRING_PROFILES_ACTIVE"])
	require.Equal(t, "-Xmx1024m -Xms512m -XX:+UseG1GC", ctx.Deploy.Variables["JAVA_OPTS"])

	// Should not have Spring Boot specific variables since it's not a Spring Boot project
	require.Empty(t, ctx.Deploy.Variables["SPRING_DEVTOOLS_RESTART_ENABLED"])
	require.Empty(t, ctx.Deploy.Variables["SPRING_DEVTOOLS_LIVERELOAD_ENABLED"])
	require.Empty(t, ctx.Deploy.Variables["SPRING_JPA_SHOW_SQL"])
}
