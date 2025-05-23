package setup

import (
	"os/exec"
	"strings"
)

func VerifyCliToolInstallations() {
	cliTools := []string{
		"docker version",
		"docker compose version",
	}

	for _, fullCmd := range cliTools {
		parts := strings.Split(fullCmd, " ")
		toolName := parts[0]
		cmdArgs := parts[1:]

		crashIfToolIsNotInstalled(toolName, cmdArgs)
	}
	logger.Info("All required CLI tools seem to be installed.")
}

func crashIfToolIsNotInstalled(toolName string, args []string) {
	cmd := exec.Command(toolName, args...)
	if err := cmd.Run(); err != nil {
		logger.Fatal("Error, tried command '%s %s' but CLI tool seems not to be installed properly.", toolName, strings.Join(args, " "))
	}
}
