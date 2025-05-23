package setup

import (
	"fmt"
	"github.com/ocelot-cloud/shared/validation"
	"ocelot/backend/apps/common"
	"ocelot/backend/security"
	"ocelot/backend/tools"
	"os"
	"time"
)

const (
	initialAdminNameEnv     = "INITIAL_ADMIN_NAME"
	initialAdminPasswordEnv = "INITIAL_ADMIN_PASSWORD"
)

func InitializeDatabase() {
	common.InitializeDatabase(tools.Config.IsUsingDockerNetwork, tools.Config.UseProductionDatabaseContainer)
	if tools.Profile == tools.PROD {
		err := createAdminUserIfNotExistent(os.Getenv(initialAdminNameEnv), os.Getenv(initialAdminPasswordEnv))
		if err != nil {
			Logger.Fatal("Failed to create admin user: %v", err)
		}
	} else {
		initializeSampleUser()
	}
}

func initializeSampleUser() {
	common.WipeWholeDatabase()
	admin := "admin"
	tools.Logger.Info("Creating sample user '%s'", admin)

	err := security.UserRepo.CreateUser(admin, "password", true)
	if err != nil {
		tools.Logger.Fatal("Failed to create sample user: %v", err)
	}
	err = security.UserRepo.SaveCookie(admin, tools.TestCookieValue, time.Now().Add(tools.CookieExpirationTime))
	if err != nil {
		tools.Logger.Error("failed to save cookie: %v", err)
	} else {
		tools.Logger.Info("Initial admin cookie for '%s' created", admin)
	}
}

func createAdminUserIfNotExistent(adminNameEnv string, adminPasswordEnv string) error {
	doesAdminExist, err := security.UserRepo.DoesAnyAdminUserExist()
	if err != nil {
		return err
	}

	if doesAdminExist {
		tools.Logger.Info("There is at least one admin user in the database, so admin initialization via env variables will not be conducted.")
		return nil
	} else {
		tools.Logger.Info("Application needs at least one admin user, but none was found in database. Trying to create the admin user from env variables.")
		return createAdminsUser(adminNameEnv, adminPasswordEnv)
	}
}

func createAdminsUser(adminNameEnv string, adminPasswordEnv string) error {
	if adminNameEnv == "" {
		return fmt.Errorf("necessary env variable '%s' is not set", initialAdminNameEnv)
	} else if adminPasswordEnv == "" {
		return fmt.Errorf("necessary env variable '%s' is not set", initialAdminPasswordEnv)
	} else {
		err := validation.ValidateStruct(tools.UserNameString{Value: adminNameEnv})
		if err != nil {
			tools.Logger.Error("admin name env variable '%s' is invalid input: %v", initialAdminNameEnv, err)
			return err
		}
		err = validation.ValidateStruct(tools.PasswordString{Value: adminPasswordEnv})
		if err != nil {
			tools.Logger.Error("admin password env variable '%s' is invalid input: %v", initialAdminPasswordEnv, err)
			return err
		}

		err = security.UserRepo.CreateUser(adminNameEnv, adminPasswordEnv, true)
		if err != nil {
			return fmt.Errorf("initial admin user creation from env variables failed: %v", err)
		}
		tools.Logger.Info("Initial admin user '%s' created", adminNameEnv)
		return nil
	}
}
