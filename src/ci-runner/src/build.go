package src

import (
	"fmt"
	"github.com/ocelot-cloud/task-runner"
	"os"
	"os/exec"
	"strings"
)

var (
	SkipBackendBuild     bool
	SkipFrontendBuild    bool
	SkipDockerImageBuild bool
	SkipIntegrationTest  bool

	sampleAppContainerName = "sampleapp"
	sampleAppImageName     = sampleAppContainerName + ":local"
)

type COMPONENT int

const (
	Backend COMPONENT = iota
	Frontend
	DockerImageLocal
)

type Component struct {
	name      string
	SkipBuild bool
	build     func()
}

var ComponentBuilds = map[COMPONENT]*Component{
	Backend: {"backend", false, func() {
		tr.ExecuteInDir(BackendDir, "go build")
	}},
	Frontend: {"frontend", false, func() {
		tr.ExecuteInDir(FrontendDir, "npm run build")
	}},
	DockerImageLocal: {"docker image", false, func() {
		// The flags make it executable in Docker containers
		tr.ExecuteInDir(BackendDir, "go build", "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
		BuildFrontend()
		tr.ExecuteInDir(ProjectDir, "docker rm -f ocelotcloud/ocelotcloud:local")

		ocelotcloudImageWithTag := getOcelotcloudImageWithTag()
		imageDownloadCmd := fmt.Sprintf("bash -c 'if [ -z \"$(docker images -q %s)\" ]; then docker pull %s; fi'", ocelotcloudImageWithTag, ocelotcloudImageWithTag)
		tr.ExecuteInDir(ProjectDir, imageDownloadCmd)

		ocelotcloudImageBuildCommand := fmt.Sprintf("docker build -t ocelotcloud/ocelotcloud:local -f %s/Dockerfile.ocelotcloud .", BackendDockerDir)
		tr.ExecuteInDir(ProjectDir, ocelotcloudImageBuildCommand)
	}},
}

func BuildFrontend() {
	tr.ExecuteInDir(FrontendDir, "npm run build")
}

func getOcelotcloudImageWithTag() string {
	data, err := os.ReadFile(BackendDockerDir + "/Dockerfile.ocelotcloud")
	if err != nil {
		tr.ColoredPrintln("Error reading Dockerfile: " + err.Error())
		tr.CleanupAndExitWithError()
	}
	line := strings.SplitN(string(data), "\n", 2)[0]
	return strings.Fields(line)[1]
}

func Build(comp COMPONENT) {
	if SkipBackendBuild {
		ComponentBuilds[Backend].SkipBuild = true
	}
	if SkipFrontendBuild {
		ComponentBuilds[Frontend].SkipBuild = true
	}
	if SkipDockerImageBuild {
		ComponentBuilds[Backend].SkipBuild = true
		ComponentBuilds[Frontend].SkipBuild = true
		ComponentBuilds[DockerImageLocal].SkipBuild = true
	}
	component := ComponentBuilds[comp]
	if component.SkipBuild {
		tr.ColoredPrintln(component.name + " build skipped")
	} else {
		component.build()
		component.SkipBuild = true
	}
}

func DownloadDependencies() {
	tr.PrintTaskDescription("downloading dependencies")
	tr.ExecuteInDir(acceptanceTestsDir, "npm install")
	tr.ExecuteInDir(FrontendDir, "npm install")
	tr.ExecuteInDir(BackendDir, "go mod tidy")
}

func BuildLocalSampleAppDockerImageIfNotPresent() {
	if !dockerSampleAppImageExists() {
		tr.ColoredPrintln("Building " + sampleAppImageName + " Docker image")
		tr.ExecuteInDir(backendSampleAppDir, "go build", "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
		tr.ExecuteInDir(backendSampleAppDir, "docker build -t "+sampleAppImageName+" .")
	} else {
		tr.ColoredPrintln(sampleAppImageName + " Docker image already exists, skipping its build")
	}
}

func dockerSampleAppImageExists() bool {
	cmd := exec.Command("docker", "images", "-q", sampleAppImageName)
	out, err := cmd.Output()
	return err == nil && len(out) > 0
}
