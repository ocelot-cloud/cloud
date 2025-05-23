package src

import (
	"os"
	"path/filepath"
	"strings"
)

const Scheme = "http"
const RootDomain = "localhost"
const ocelotUrl = Scheme + "://ocelot-cloud." + RootDomain
const frontendServerUrl = Scheme + "://localhost:8081"

func getDockerCommand(profile, imageTag string, isRemoteDeployment bool) string {
	var runDetachedString string
	if isRemoteDeployment {
		runDetachedString = "-d"
	}

	baseCommand := []string{
		"docker run",
		runDetachedString,
		"--rm",
		"--name ocelotcloud",
		"-p 80:8080",
		"-p 443:8443",
		"-e PROFILE=" + profile,
		"-e LOG_LEVEL=DEBUG",
		"-e INITIAL_ADMIN_NAME=admin",
		"-e INITIAL_ADMIN_PASSWORD=password",
		"-v /tmp/ocelotcloud:/tmp/ocelotcloud",
		"-v /var/run/docker.sock:/var/run/docker.sock",
	}

	return strings.Join(baseCommand, " ") + " ocelotcloud/ocelotcloud:" + imageTag
}

const cypressCommand = "npx cypress run --spec cypress/e2e/cloud.cy.ts --headless"

var (
	ProjectDir = GetProjectDir()
	srcDir     = ProjectDir + "/src"

	BackendDir         = srcDir + "/backend"
	ciRunnerDir        = srcDir + "/ci-runner"
	FrontendDir        = srcDir + "/frontend"
	acceptanceTestsDir = srcDir + "/cypress"

	backendAppsDir     = BackendDir + "/apps"
	backendAppStoreDir = backendAppsDir + "/store"

	assetsDir                = BackendDir + "/assets"
	BackendDockerDir         = assetsDir + "/docker"
	backendSampleAppDir      = BackendDockerDir + "/sampleapp"
	backendComponentTestsDir = BackendDir + "/component-tests"

	appStoreSrcDir      = ProjectDir + "/../store/src"
	appStoreCiRunnerDir = appStoreSrcDir + "/ci-runner"
	appStoreBackendDir  = appStoreSrcDir + "/backend"
)

const NativeProfile = "NATIVE"
const DockerTestProfile = "DOCKER_TEST"
const ProdProfile = "PROD"

func GetProjectDir() string {
	ciRunnerDirectory, _ := os.Getwd()
	src := filepath.Dir(ciRunnerDirectory)
	return filepath.Dir(src)
}

func GetProfileEnv(profile string) string {
	return "PROFILE=" + profile
}
