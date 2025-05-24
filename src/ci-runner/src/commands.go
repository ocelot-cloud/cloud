package src

import (
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/task-runner"
	"net"
	"os"
	"os/exec"
	"time"
)

func TestUnitsFast() {
	tr.PrintTaskDescription("Executing backend unit tests fast")
	defer tr.Cleanup()
	tr.ExecuteInDir(BackendDir, "go test -p 1 -v -count=1 -tags fast ./...")
}

func TestUnitsSlow() {
	tr.PrintTaskDescription("Executing backend unit tests slow")
	defer tr.Cleanup()
	tr.ExecuteInDir(BackendDir, "go test -p 1 -v -count=1 -tags slow ./...")
}

func TestNativelyRunningBackend() {
	tr.PrintTaskDescription("Testing backend component")
	defer tr.Cleanup()
	tr.ExecuteInDir(BackendDir, "rm -rf data")
	Build(Backend)
	tr.StartDaemon(BackendDir, "./backend", GetProfileEnv(NativeProfile), "USE_REAL_SSH_CLIENT=true")
	tr.WaitUntilPortIsReady("8080")
	tr.ExecuteInDir(backendComponentTestsDir, "go test -p 1 -v -count=1 -tags component ./...", GetProfileEnv(NativeProfile))
}

func TestNativelyRunningBackendMocked() {
	tr.PrintTaskDescription("Testing mocked backend component")
	defer tr.Cleanup()
	tr.ExecuteInDir(BackendDir, "rm -rf data")
	Build(Backend)
	tr.StartDaemon(BackendDir, "./backend", GetProfileEnv(NativeProfile), "USE_MOCKS=true")
	tr.WaitUntilPortIsReady("8080")
	tr.ExecuteInDir(backendComponentTestsDir, "go test -p 1 -v -count=1 -tags component ./...", GetProfileEnv(NativeProfile))
}

func DeployContainer(profile string) {
	tr.Execute("docker rm -f ocelotcloud ocelotcloud_ocelotdb_ocelotdb")
	tr.Execute("docker volume rm -f ocelotcloud_ocelotdb_data")
	Build(DockerImageLocal)
	tr.StartDaemon(".", getDockerCommand(profile, "local", false))
	tr.WaitForWebPageToBeReady(ocelotUrl)
	if profile == ProdProfile {
		shouldPostgresPortBeAvailableViaLocalhost(false)
	} else {
		shouldPostgresPortBeAvailableViaLocalhost(true)
	}
}

func shouldPostgresPortBeAvailableViaLocalhost(shouldPortBeAvailable bool) {
	address := "localhost:5432"
	timeout := 2 * time.Second
	_, err := net.DialTimeout("tcp", address, timeout)

	if shouldPortBeAvailable {
		if err == nil {
			tr.ColoredPrintln("Postres port is available as expected via localhost")
		} else {
			tr.ColoredPrintln("Postres port is unexpectedly unavailable via localhost, should not be for debugging reasons")
			tr.CleanupAndExitWithError()
		}
	} else {
		if err == nil {
			tr.ColoredPrintln("Postres port is unexpectedly available via localhost, should not be for security reasons")
			tr.CleanupAndExitWithError()
		} else {
			tr.ColoredPrintln("Postres port is unavailable as expected via localhost")
		}
	}
}

func TestAll() {
	tr.PrintTaskDescription("Running all CI tests")
	TestsFast()
	TestsSlow()
}

func TestsSlow() {
	TestUnitsSlow()
	TestNativelyRunningBackend()
	TestDockerTestContainer()
	TestDockerProdContainer()
}

func TestDockerProdContainer() {
	tr.PrintTaskDescription("Testing docker prod container")
	defer tr.Cleanup()
	DeployContainer(ProdProfile)
	tr.ExecuteInDir(backendComponentTestsDir, "go test -p 1 -v -count=1 -tags prod_container ./...", GetProfileEnv(ProdProfile))
}

func TestsFast() {
	TestUnitsFast()
	TestNativelyRunningBackendMocked()
	TestFrontend()
}

func TestIntegration() {
	if SkipIntegrationTest {
		tr.PrintTaskDescription("Skipping integration tests")
		return
	}
	tr.PrintTaskDescription("Testing integration between cloud and app store")
	defer tr.Cleanup()
	tr.ExecuteInDir(appStoreCiRunnerDir, "docker compose up -d")
	tr.ExecuteInDir(appStoreBackendDir, "go build")
	tr.StartDaemon(appStoreBackendDir, "bash run-dev.sh")
	tr.WaitUntilPortIsReady("8082")
	tr.ExecuteInDir(backendAppStoreDir, "go test -p 1 -tags integration -v -count=1 ./...", GetProfileEnv(NativeProfile))
}

func TestDockerTestContainer() {
	tr.PrintTaskDescription("Testing docker test container")
	defer tr.Cleanup()
	DeployContainer(DockerTestProfile)
	tr.ExecuteInDir(backendComponentTestsDir, "go test -p 1 -v -count=1 -tags component ./...", GetProfileEnv(DockerTestProfile))
}

func TestFrontend() {
	tr.PrintTaskDescription("Testing Components In DevelopmentMode")
	defer tr.Cleanup()
	Build(Backend)
	tr.StartDaemon(BackendDir, "./backend", GetProfileEnv(NativeProfile), "INITIAL_ADMIN_NAME=admin", "INITIAL_ADMIN_PASSWORD=password", "USE_MOCKS=true")
	tr.WaitUntilPortIsReady("8080")

	Build(Frontend)
	tr.StartDaemon(FrontendDir, "npm run serve", "VITE_APP_PROFILE="+NativeProfile)
	tr.WaitForWebPageToBeReady(frontendServerUrl)
	tr.ExecuteInDir(acceptanceTestsDir, cypressCommand)
}

func UpdateDependencies() {
	tr.ExecuteInDir(BackendDir, "go get -u ./...")
	tr.ExecuteInDir(BackendDir, "go mod tidy")
	tr.ExecuteInDir(BackendDir, "go build")

	tr.ExecuteInDir(ciRunnerDir, "go get -u ./...")
	tr.ExecuteInDir(ciRunnerDir, "go mod tidy")
	tr.ExecuteInDir(ciRunnerDir, "go build")

	tr.ExecuteInDir(FrontendDir, "yarn upgrade --latest --pattern \"*\"")
	tr.ExecuteInDir(FrontendDir, "yarn add vue@latest vite@latest")
	tr.ExecuteInDir(FrontendDir, "yarn install")
	tr.ExecuteInDir(FrontendDir, "yarn build")

	tr.ExecuteInDir(acceptanceTestsDir, "npx npm-check-updates -u")
	tr.ExecuteInDir(acceptanceTestsDir, "npm install cypress@latest")
	tr.ExecuteInDir(acceptanceTestsDir, "npm install")

	tr.ExecuteInDir(BackendDockerDir, "./update-ocelotcloud-docker-image-tag.sh Dockerfile.ocelotcloud")
	tr.ExecuteInDir(BackendDockerDir, "./update-ocelotcloud-docker-image-tag.sh Dockerfile.restic")
}

var DemoServerName = "demo"
var TestServerName = "test"

func DeployToServer(server string) {
	tr.PrintTaskDescription("Deploying to server: " + server)
	executeCommandOnServer(server, "docker ps -aq | xargs -r docker rm -f")
	executeCommandOnServer(server, "docker volume prune -af")
	executeCommandOnServer(server, "docker network prune -f")
	executeCommandOnServer(server, "docker pull ocelotcloud/ocelotcloud:demo")
	demoDockerCommand := getDockerCommand(ProdProfile, "demo", true)
	executeCommandOnServer(server, demoDockerCommand)
}

func executeCommandOnServer(server, command string) {
	tr.ColoredPrintln("Executing command on server '%s': %s", server, command)
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("ssh %s '%s'", server, command))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		utils.Logger.Fatal("Error while executing command on server: %v", err)
	}
}
