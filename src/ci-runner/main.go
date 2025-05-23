package main

import (
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/task-runner"
	"github.com/spf13/cobra"
	"net"
	"ocelot/ci-runner/src"
	"os"
	"os/exec"
	"strings"
)

var (
	ocelotDbContainerName = "ocelotcloud_ocelotdb_ocelotdb"
)

var rootCmd = &cobra.Command{
	Use:   "ci-runner",
	Short: "CI Runner CLI",
	Long:  `CI Runner CLI to build, test, and deploy projects.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build docker image",
	Long:  "Builds the whole project from scratch and produces a production docker image",
	Run: func(cmd *cobra.Command, args []string) {
		src.Build(src.DockerImageLocal)
		tr.ColoredPrintln("\nSuccess! Build worked.\n")
	},
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Removes processes and docker artifacts",
	Long:  "Removes processes and docker artifacts",
	Run: func(cmd *cobra.Command, args []string) {
		tr.Cleanup()
		tr.ColoredPrintln("\nSuccess! Cleanup worked.\n")
	},
}

var fullCleanupCmd = &cobra.Command{
	Use:   "full-cleanup",
	Short: "Removes processes and docker artifacts",
	Long:  "Removes processes and docker artifacts",
	Run: func(cmd *cobra.Command, args []string) {
		tr.Cleanup()
		removeContainersCommand := fmt.Sprintf("docker rm -f ocelotcloud $s restic remote_backup_server || true'", ocelotDbContainerName)
		tr.Execute(removeContainersCommand)
		tr.Execute("docker rmi -f ocelotcloud/ocelotcloud:local restic:local sampleapp:local app:local || true")
		tr.Execute("docker network rm $(docker network ls -q)")
		tr.Execute("docker volume rm $(docker volume ls -q)")
		tr.ColoredPrintln("\nSuccess! Cleanup worked.\n")
	},
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run various tests",
	Long:  "Run different types of tests for cloud, hub, ci, or schedule.",
}

var downloadDependenciesCmd = &cobra.Command{
	Use:   "download",
	Short: "Downloads application dependencies",
	Long:  "Downloads all necessary dependencies for the application. This step must be performed once at the beginning of development to set up the environment. This process is separated from other commands so that they do not check dependencies on each run, making them faster.",
	Run: func(cmd *cobra.Command, args []string) {
		src.DownloadDependencies()
	},
}

var testFrontendCmd = &cobra.Command{
	Use:   "frontend",
	Short: "tests frontend",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestFrontend()
	},
}

var testBackendUnitsFastCmd = &cobra.Command{
	Use:   "units-fast",
	Short: "test backend units fast",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestUnitsFast()
	},
}

var testBackendUnitsSlowCmd = &cobra.Command{
	Use:   "units-slow",
	Short: "test backend units slow",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestUnitsSlow()
	},
}

var testFastCmd = &cobra.Command{
	Use:   "fast",
	Short: "only runs fast tests",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestsFast()
	},
}

var testSlowCmd = &cobra.Command{
	Use:   "slow",
	Short: "only runs slow tests",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestsSlow()
	},
}

var testBackendNativeCmd = &cobra.Command{
	Use:   "native",
	Short: "test backend api natively running backend",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestNativelyRunningBackend()
	},
}

var testBackendNativeMockedCmd = &cobra.Command{
	Use:   "native-mocked",
	Short: "test backend api natively running backend with mocks",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestNativelyRunningBackendMocked()
	},
}

var testBackendDockerTestCmd = &cobra.Command{
	Use:   "docker-test",
	Short: "test backend API of docker test container",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestDockerTestContainer()
	},
}

var testBackendDockerProdCmd = &cobra.Command{
	Use:   "docker-prod",
	Short: "test backend API of docker prod container",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestDockerProdContainer()
	},
}

var testIntegrationCmd = &cobra.Command{
	Use:   "integration",
	Short: "test integration between backend and app store",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestIntegration()
	},
}

var testAllCmd = &cobra.Command{
	Use:   "all",
	Short: "test everything",
	Run: func(cmd *cobra.Command, args []string) {
		src.TestAll()
	},
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the ocelot-cloud docker container",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var deployLocalTestCmd = &cobra.Command{
	Use:   "local-test",
	Short: "Deploy the ocelot-cloud docker container locally with profile DOCKER_TEST",
	Run: func(cmd *cobra.Command, args []string) {
		tr.PrintTaskDescription("Running a docker test server")
		src.DeployContainer(src.DockerTestProfile)
	},
}

var deployLocalProdCmd = &cobra.Command{
	Use:   "local-prod",
	Short: "Deploy the ocelot-cloud docker container locally with profile PROD",
	Run: func(cmd *cobra.Command, args []string) {
		tr.PrintTaskDescription("Running a production server")
		src.DeployContainer(src.ProdProfile)
	},
}

var deployDemoCmd = &cobra.Command{
	Use:   "remote-demo",
	Short: "Deploy the ocelot-cloud docker container to the demo server",
	Run: func(cmd *cobra.Command, args []string) {
		tr.PrintTaskDescription("Deploying demo to production server")
		tr.PromptForContinuation("Are you sure you want to deploy to the demo server?")
		src.DeployToServer(src.DemoServerName)
	},
}

var deployTestCmd = &cobra.Command{
	Use:   "remote-test",
	Short: "Deploy the ocelot-cloud docker container to the test server",
	Run: func(cmd *cobra.Command, args []string) {
		tr.PrintTaskDescription("Deploying test to production server")
		src.DeployToServer(src.TestServerName)
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "updates dependencies",
	Run: func(cmd *cobra.Command, args []string) {
		src.UpdateDependencies()
	},
}

var analyseCodeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Runs code analysis tools on the backend and fails if there are any issues",
	Run: func(cmd *cobra.Command, args []string) {
		utils.AnalyzeCode(src.BackendDir)
	},
}

func main() {
	tr.HandleSignals()
	tr.DefaultEnvs = []string{"LOG_LEVEL=DEBUG"}
	tr.CustomCleanupFunc = CustomCleanup

	rootCmd.Root().CompletionOptions.DisableDefaultCmd = true
	pf := rootCmd.PersistentFlags()
	pf.BoolVarP(&src.SkipBackendBuild, "skip-backend-build", "b", false, "Skip building the backend")
	pf.BoolVarP(&src.SkipFrontendBuild, "skip-frontend-build", "f", false, "Skip building the frontend")
	pf.BoolVarP(&src.SkipDockerImageBuild, "skip-docker-build", "d", false, "Skip building the Docker container, including skipping building the backend and frontend")
	pf.BoolVarP(&src.SkipIntegrationTest, "skip-integration", "i", false, " If called, skips the integration tests")

	src.ComponentBuilds[src.Backend].SkipBuild = src.SkipBackendBuild
	src.ComponentBuilds[src.Frontend].SkipBuild = src.SkipFrontendBuild
	src.ComponentBuilds[src.DockerImageLocal].SkipBuild = src.SkipDockerImageBuild

	testCmd.AddCommand(testAllCmd, testBackendNativeCmd, testBackendNativeMockedCmd, testFrontendCmd, testBackendDockerTestCmd, testBackendDockerProdCmd, testIntegrationCmd, testBackendUnitsFastCmd, testBackendUnitsSlowCmd, testFastCmd, testSlowCmd)
	deployCmd.AddCommand(deployLocalTestCmd, deployLocalProdCmd, deployDemoCmd, deployTestCmd)
	rootCmd.AddCommand(buildCmd, testCmd, deployCmd, cleanCmd, fullCleanupCmd, downloadDependenciesCmd, updateCmd, analyseCodeCmd, uploadCmd)

	if shouldDoPreChecks() {
		tr.Cleanup()
		failIfRequiredPortsAreAlreadyInUse()
		failIfThereAreExistingDockerContainers()
		src.BuildLocalSampleAppDockerImageIfNotPresent()
	}

	if err := rootCmd.Execute(); err != nil {
		tr.ColoredPrintln("\nError during execution: %s\n", err.Error())
		tr.CleanupAndExitWithError()
	}
}

func shouldDoPreChecks() bool {
	if len(os.Args) == 1 {
		return false
	} else if len(os.Args) > 1 && (os.Args[1] == "completion" || os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help") {
		return false
	} else {
		return true
	}
}

func failIfRequiredPortsAreAlreadyInUse() {
	ports := []string{"8080", "8081", "8082"}

	for _, port := range ports {
		listener, err := net.Listen("tcp", ":"+port)
		if err != nil {
			fmt.Printf("Error: Port %s is already in use. Exiting.\n", port)
			os.Exit(1)
		} else {
			err := listener.Close()
			if err != nil {
				fmt.Printf("Could not close listener on port %s.\n", port)
				os.Exit(1)
			}
			fmt.Printf("Port %s is available.\n", port)
		}
	}
}

func failIfThereAreExistingDockerContainers() {
	containers := GetRunningContainers()
	for _, container := range containers {
		if container != ocelotDbContainerName && container != "" {
			fmt.Println("Error: There are existing Docker containers. Please destroy them and try again.")
			os.Exit(1)
		}
	}

	fmt.Printf("As required for DevOps jobs, no Docker containers are deployed (except '%s').\n", ocelotDbContainerName)
}

func GetRunningContainers() []string {
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error executing docker command: %v\n", err)
		os.Exit(1)
	}
	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	return containers
}

var potentiallyPreExistingProcesses = []string{
	"./backend",
	"vue-tr-service",
	"vue-service",
	"vite",
}

func CustomCleanup() {
	tr.KillProcesses(potentiallyPreExistingProcesses)
	dockerPruningCommands := []string{
		"docker network prune -f",
		"docker volume prune -a -f",
		"docker image prune -f"}
	for _, command := range dockerPruningCommands {
		tr.Execute(command)
	}
	containers := GetRunningContainers()
	for _, container := range containers {
		if container != ocelotDbContainerName && container != "" {
			tr.Execute("docker rm -f " + container)
		}
	}

	tr.ExecuteInDir(src.BackendDir, "rm -rf data")
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload the ocelotcloud docker image",
	Run: func(cmd *cobra.Command, args []string) {
		var inputTag string
		fmt.Print("Enter tag of image to upload: ")
		_, err := fmt.Scanln(&inputTag)
		if err != nil {
			fmt.Println("Error:", err)
			tr.CleanupAndExitWithError()
		}

		src.Build(src.DockerImageLocal)

		tagImageAndUpload(inputTag)
		if inputTag != "demo" {
			uploadExtraTag(inputTag)
		}
	},
}

func uploadExtraTag(inputTag string) {
	if isBetaTag(inputTag) {
		tagImageAndUpload("beta")
	} else {
		tagImageAndUpload("stable")
	}
}

func isBetaTag(inputTag string) bool {
	return strings.HasPrefix(inputTag, "0.") || strings.HasPrefix(inputTag, "v0.")
}

func tagImageAndUpload(inputTag string) {
	tr.Execute("docker tag ocelotcloud/ocelotcloud:local ocelotcloud/ocelotcloud:" + inputTag)
	tr.Execute("docker push ocelotcloud/ocelotcloud:" + inputTag)
}
