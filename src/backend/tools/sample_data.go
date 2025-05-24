package tools

import (
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/task-runner"
	"os"
	"path/filepath"
	"time"
)

var (
	SampleMaintainer         = "samplemaintainer"
	SampleApp                = "sampleapp"
	SampleAppDockerContainer = SampleMaintainer + "_" + SampleApp + "_" + SampleApp
	SampleAppDockerNetwork   = SampleMaintainer + "_" + SampleApp
	SampleAppDockerVolume    = SampleMaintainer + "_" + SampleApp + "_data"

	SampleAppCreationTimestamp = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	SampleAppId = "1"

	SampleAppVersion1Id                = "11"
	SampleAppVersion1Name              = "1.0"
	SampleAppVersion1CreationTimestamp = SampleAppCreationTimestamp.Add(-time.Hour)
	SampleAppVersion2Id                = "12"
	SampleAppVersion2Name              = "2.0"
	SampleAppVersion2CreationTimestamp = SampleAppCreationTimestamp.Add(+time.Hour)

	SampleBackupDescription BackupDescription = "sample-description"
)

var sampleAppContent []byte = nil

func GetSampleAppContent() []byte {
	if sampleAppContent == nil {
		sampleAppContent = CreateTempZippedApp(
			"/"+SampleApp,
			"docker-compose.yml",
			false,
			SampleAppDir,
			"app.yml",
		)
	}
	return sampleAppContent
}

func CreateTempZippedApp(
	appSubfolder,
	dockerComposeFileName string,
	renameComposeToDefault bool,
	sampleFolder string,
	extraCopyFiles ...string,
) []byte {
	err := os.MkdirAll(OcelotCloudTempDir, 0700)
	if err != nil {
		Logger.Fatal("Failed to create temp dir for sample app: %v", err)
	}
	sampleAppTempDir, err := os.MkdirTemp(OcelotCloudTempDir, filepath.Base(appSubfolder))
	if err != nil {
		Logger.Fatal("Failed to create temp dir for sample app: %v", err)
	}
	defer utils.RemoveDir(sampleAppTempDir)

	for _, file := range extraCopyFiles {
		tr.Copy(sampleFolder, file, sampleAppTempDir)
	}

	tr.Copy(sampleFolder, dockerComposeFileName, sampleAppTempDir)

	updatedDockerComposeYamlName := "docker-compose-updated.yml"
	if renameComposeToDefault && dockerComposeFileName == updatedDockerComposeYamlName {
		tr.Rename(sampleAppTempDir, updatedDockerComposeYamlName, "docker-compose.yml")
	}

	sampleAppContent, err := utils.ZipDirectoryToBytes(sampleAppTempDir)
	if err != nil {
		Logger.Fatal("Failed to zip sample app: %v", err)
	}

	return sampleAppContent
}

var (
	OcelotDbMaintainer  = "ocelotcloud"
	OcelotDbAppName     = "ocelotdb"
	ResticAppName       = "restic"
	ResticDockerNetwork = OcelotDbMaintainer + "_" + ResticAppName
)
