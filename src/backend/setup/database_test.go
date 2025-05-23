//go:build fast

package setup

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/apps/common"
	"ocelot/backend/security"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	common.InitializeDatabase(false, false)
	common.WipeWholeDatabase()
	defer common.WipeWholeDatabase()
	m.Run()
}

func TestEmptyAdminInitializationEnvsShouldFail(t *testing.T) {
	defer common.WipeWholeDatabase()
	assert.NotNil(t, createAdminUserIfNotExistent("", ""))
	assert.NotNil(t, createAdminUserIfNotExistent("admin", ""))
	assert.NotNil(t, createAdminUserIfNotExistent("", "password"))
}

func TestAdminInitializationWithCorrectEnvs(t *testing.T) {
	defer common.WipeWholeDatabase()
	assert.Nil(t, createAdminUserIfNotExistent("admin", "password"))
}

func TestAdminInitializationIsIgnoredWhenAlreadyExistsInDatabase(t *testing.T) {
	defer common.WipeWholeDatabase()
	err := security.UserRepo.CreateUser("admin", "password", true)
	assert.Nil(t, err)
	assert.Nil(t, createAdminUserIfNotExistent("", ""))
}

func TestDefaultAdminCreation(t *testing.T) {
	defer common.WipeWholeDatabase()
	doesExist, err := security.UserRepo.DoesAnyAdminUserExist()
	assert.Nil(t, err)
	assert.False(t, doesExist)

	assert.Nil(t, createAdminUserIfNotExistent("admin", "password"))
	doesExist, err = security.UserRepo.DoesAnyAdminUserExist()
	assert.Nil(t, err)
	assert.True(t, doesExist)
	assert.True(t, security.UserRepo.IsPasswordCorrect("admin", "password"))

	assert.Nil(t, security.UserRepo.SaveCookie("admin", "some-cookie", time.Now()))
	auth, err := security.UserRepo.GetAuthenticationViaCookie("some-cookie")
	assert.Nil(t, err)
	assert.Equal(t, "admin", auth.User)
	assert.True(t, auth.IsAdmin)
}
