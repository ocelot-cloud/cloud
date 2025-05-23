//go:build fast

package security

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/apps/common"
	"ocelot/backend/tools"
	"reflect"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	common.InitializeDatabase(false, false)
	defer common.WipeWholeDatabase()
	m.Run()
}

var (
	sampleUser     = "user"
	samplePassword = "password"
)

func TestPasswordCorrectness(t *testing.T) {
	defer common.WipeWholeDatabase()
	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, false))
	assert.True(t, UserRepo.IsPasswordCorrect(sampleUser, samplePassword))
	assert.False(t, UserRepo.IsPasswordCorrect(sampleUser, samplePassword+"x"))

	assert.NotNil(t, UserRepo.CreateUser(sampleUser, samplePassword+"x", false))

	userId, err := UserRepo.GetUserId(sampleUser)
	assert.Nil(t, err)
	assert.Nil(t, UserRepo.DeleteUser(userId))
	assert.False(t, UserRepo.IsPasswordCorrect(sampleUser, samplePassword))
}

func TestDoesUserExist(t *testing.T) {
	defer common.WipeWholeDatabase()
	assert.False(t, UserRepo.DoesUserExist(sampleUser))
	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, false))
	assert.True(t, UserRepo.DoesUserExist(sampleUser))
	userId, err := UserRepo.GetUserId(sampleUser)
	assert.Nil(t, err)
	assert.Nil(t, UserRepo.DeleteUser(userId))
	assert.False(t, UserRepo.DoesUserExist(sampleUser))
}

func TestGetUserWithCookie(t *testing.T) {
	defer common.WipeWholeDatabase()
	_, err := UserRepo.GetAuthenticationViaCookie(tools.TestCookieValue)
	assert.NotNil(t, err)

	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, false))
	userId, err := UserRepo.GetUserId(sampleUser)
	assert.Nil(t, err)
	cookieExpirationDate := time.Date(2023, 10, 10, 10, 10, 10, 0, time.UTC)
	assert.Nil(t, UserRepo.SaveCookie(sampleUser, tools.TestCookieValue, cookieExpirationDate))
	auth, err := UserRepo.GetAuthenticationViaCookie(tools.TestCookieValue)
	assert.Nil(t, err)
	assert.Equal(t, userId, auth.UserId)
	assert.Equal(t, sampleUser, auth.User)
	assert.True(t, auth.CookieExpirationDate.Equal(cookieExpirationDate))
	assert.False(t, auth.IsAdmin)
	common.WipeWholeDatabase()

	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, true))
	assert.Nil(t, UserRepo.SaveCookie(sampleUser, tools.TestCookieValue, time.Now()))
	auth, err = UserRepo.GetAuthenticationViaCookie(tools.TestCookieValue)
	assert.Nil(t, err)
	assert.Equal(t, sampleUser, auth.User)
	assert.True(t, auth.IsAdmin)
}

func TestDoesAnyAdminUserExist(t *testing.T) {
	defer common.WipeWholeDatabase()
	doesExist, err := UserRepo.DoesAnyAdminUserExist()
	assert.Nil(t, err)
	assert.False(t, doesExist)

	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, false))
	doesExist, err = UserRepo.DoesAnyAdminUserExist()
	assert.Nil(t, err)
	assert.False(t, doesExist)

	assert.Nil(t, UserRepo.CreateUser(sampleUser+"x", samplePassword, true))
	doesExist, err = UserRepo.DoesAnyAdminUserExist()
	assert.Nil(t, err)
	assert.True(t, doesExist)
}

func TestLogout(t *testing.T) {
	defer common.WipeWholeDatabase()
	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, false))
	assert.Nil(t, UserRepo.SaveCookie(sampleUser, tools.TestCookieValue, time.Now()))
	auth, err := UserRepo.GetAuthenticationViaCookie(tools.TestCookieValue)
	assert.Nil(t, err)
	assert.Equal(t, sampleUser, auth.User)

	assert.Nil(t, UserRepo.Logout(sampleUser))
	assert.True(t, UserRepo.DoesUserExist(sampleUser))
	_, err = UserRepo.GetAuthenticationViaCookie(tools.TestCookieValue)
	assert.NotNil(t, err)
}

func TestChangePassword(t *testing.T) {
	defer common.WipeWholeDatabase()
	oldPassword := samplePassword
	newPassword := samplePassword + "x"
	assert.Nil(t, UserRepo.CreateUser(sampleUser, oldPassword, false))
	assert.True(t, UserRepo.IsPasswordCorrect(sampleUser, oldPassword))

	id, err := UserRepo.GetUserId(sampleUser)
	assert.Nil(t, err)
	assert.Nil(t, UserRepo.ChangePassword(id, newPassword))
	assert.False(t, UserRepo.IsPasswordCorrect(sampleUser, oldPassword))
	assert.True(t, UserRepo.IsPasswordCorrect(sampleUser, newPassword))
}

func TestSecretRandomness(t *testing.T) {
	defer common.WipeWholeDatabase()
	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, false))
	secret, _ := UserRepo.GenerateSecret(sampleUser)
	secret2, _ := UserRepo.GenerateSecret(sampleUser)
	assert.NotEqual(t, secret, secret2)
}

func TestSecretValidation(t *testing.T) {
	defer common.WipeWholeDatabase()
	cookieFromDb, err := UserRepo.GetAssociatedCookieValueAndDeleteSecret("invalid")
	assert.NotNil(t, err)
	assert.Equal(t, "", cookieFromDb)

	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, false))
	assert.Nil(t, UserRepo.SaveCookie(sampleUser, tools.TestCookieValue, time.Now()))

	secret, err := UserRepo.GenerateSecret(tools.TestCookieValue)
	assert.Nil(t, err)
	assert.Equal(t, 64, len(secret))
	cookieFromDb, err = UserRepo.GetAssociatedCookieValueAndDeleteSecret(secret)
	assert.Nil(t, err)
	assert.Equal(t, tools.TestCookieValue, cookieFromDb)

	cookieFromDb, err = UserRepo.GetAssociatedCookieValueAndDeleteSecret(secret)
	assert.NotNil(t, err)
	assert.Equal(t, "", cookieFromDb)
}

func TestListUsers(t *testing.T) {
	common.WipeWholeDatabase()
	defer common.WipeWholeDatabase()
	users, err := UserRepo.ListUsers()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(users))

	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, false))
	users, err = UserRepo.ListUsers()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
	userId, err := UserRepo.GetUserId(sampleUser)
	assert.Nil(t, err)
	assert.Equal(t, userId, users[0].Id)
	assert.Equal(t, sampleUser, users[0].Name)
	assert.False(t, users[0].IsAdmin)

	sampleUser2 := sampleUser + "x"
	assert.Nil(t, UserRepo.CreateUser(sampleUser2, samplePassword, true))
	users, err = UserRepo.ListUsers()
	assert.Nil(t, err)
	assert.Equal(t, 2, len(users))
	userId, err = UserRepo.GetUserId(sampleUser2)
	assert.Nil(t, err)
	assert.Equal(t, userId, users[1].Id)
	assert.Equal(t, sampleUser2, users[1].Name)
	assert.True(t, users[1].IsAdmin)
}

func TestTotalUserOperation(t *testing.T) {
	defer common.WipeWholeDatabase()
	cookieVal := "some-cookie"
	cookieExp := "expiration-date"
	user := tools.UserFullInfo{
		Id:                   2,
		UserName:             "sampleuser",
		HashedPassword:       "123456",
		HashedCookieValue:    &cookieVal,
		CookieExpirationDate: &cookieExp,
		IsAdmin:              true,
	}
	err := UserRepo.DeleteUsersAndAddUsersFullInfo([]tools.UserFullInfo{user})
	assert.Nil(t, err)

	users, err := UserRepo.GetAllUsersFullInfo()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
	reflect.DeepEqual(user, users[0])
}

func TestUpdateCookieExpirationDate(t *testing.T) {
	defer common.WipeWholeDatabase()
	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, false))
	cookieExpirationDate := time.Date(2023, 10, 10, 10, 10, 10, 0, time.UTC)
	assert.Nil(t, UserRepo.SaveCookie(sampleUser, tools.TestCookieValue, cookieExpirationDate))
	assert.Nil(t, UserRepo.UpdateCookieExpirationDate(tools.TestCookieValue))

	auth, err := UserRepo.GetAuthenticationViaCookie(tools.TestCookieValue)
	assert.Nil(t, err)
	assert.True(t, time.Now().Add(tools.CookieExpirationTime).Add(-1*time.Minute).Before(auth.CookieExpirationDate))
	assert.True(t, time.Now().Add(tools.CookieExpirationTime).Add(+1*time.Minute).After(auth.CookieExpirationDate))
}

func TestIsCookieExpired(t *testing.T) {
	defer common.WipeWholeDatabase()
	assert.Nil(t, UserRepo.CreateUser(sampleUser, samplePassword, false))

	cookieExpirationDate := time.Date(2023, 10, 10, 10, 10, 10, 0, time.UTC)
	assert.Nil(t, UserRepo.SaveCookie(sampleUser, tools.TestCookieValue, cookieExpirationDate))
	isExpired, err := UserRepo.IsExpired(tools.TestCookieValue)
	assert.Nil(t, err)
	assert.True(t, isExpired)

	assert.Nil(t, UserRepo.SaveCookie(sampleUser, tools.TestCookieValue, time.Now().Add(1*time.Minute)))
	isExpired, err = UserRepo.IsExpired(tools.TestCookieValue)
	assert.Nil(t, err)
	assert.False(t, isExpired)
}
