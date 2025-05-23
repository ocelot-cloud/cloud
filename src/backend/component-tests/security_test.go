//go:build component

package component_tests

import (
	"github.com/ocelot-cloud/shared/assert"
	"github.com/ocelot-cloud/shared/utils"
	"ocelot/backend/security"
	"ocelot/backend/tools"
	"testing"
)

func TestSecretGeneration(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	secret, err := cloud.getSecret()
	assert.Nil(t, err)
	assert.Equal(t, 64, len(secret))

	secret2, _ := cloud.getSecret()
	assert.NotEqual(t, secret, secret2)
}

func TestOriginPolicyActive(t *testing.T) {
	if tools.Config.AreCrossOriginRequestsAllowed {
		t.Skip()
		return
	}

	client := getClient(t)
	client.parent.Origin = "http://other-domain.com"
	_, err := client.parent.DoRequest("/api/hello", nil, "")
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, security.CrossRequestsToOcelotCloudNotAllowedErrorMessage), err.Error())

	client = getClient(t)
	client.parent.Origin = "other-domain.com"
	_, err = client.parent.DoRequest("/api/hello", nil, "")
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "invalid origin header: other-domain.com"), err.Error())

}

func TestLoginHandlerSecurity(t *testing.T) {
	cloud := getClient(t)
	defer cloud.wipeData()
	err := cloud.login()
	assert.Nil(t, err)

	workingName := cloud.parent.User
	workingPassword := cloud.parent.Password

	cloud.parent.User = workingName
	cloud.parent.Password = "wrongpassword"
	err = cloud.login()
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(401, "Invalid username or password"), err.Error())

	cloud.parent.User = "wrong"
	cloud.parent.Password = workingPassword
	err = cloud.login()
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(401, "Invalid username or password"), err.Error())
}

func TestLoginHandlerInputValidation(t *testing.T) {
	cloud := getClient(t)
	defer cloud.wipeData()
	err := cloud.login()
	assert.Nil(t, err)

	workingName := cloud.parent.User
	workingPassword := cloud.parent.Password

	cloud.parent.User = workingName
	cloud.parent.Password = "short"
	err = cloud.login()
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "invalid input"), err.Error())

	cloud.parent.User = "user!@#$"
	cloud.parent.Password = workingPassword
	err = cloud.login()
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "invalid input"), err.Error())
}

func TestUserCreationHandler(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	assert.Nil(t, cloud.createUser("testuser", "testpassword"))
	err := cloud.createUser("testuser", "testpassword")
	assert.Equal(t, utils.GetErrMsg(409, "user already exists"), err.Error())

	err = cloud.createUser("test!", "testpassword")
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "invalid input"), err.Error())

	err = cloud.createUser("testuser", "short")
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "invalid input"), err.Error())
}

func TestCookieValidation(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	cloud.parent.Cookie.Value += "a"

	err := cloud.createUser("testuser", "testpassword")
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "invalid input"), err.Error())
}

func TestSecretIsDeletedAfterExchangeAgainstCookie(t *testing.T) {
	if tools.Profile != tools.DOCKER_TEST {
		t.Skip()
		return
	}

	cloud := getClient(t)
	defer cloud.wipeData()
	assert.Nil(t, cloud.login())
	app, err := cloud.installSampleApp("2.0")
	assert.Nil(t, err)
	assert.Nil(t, cloud.startApp(app.AppId))
	cloud.setHostValue("localhost")
	secret, err := cloud.getSecret()
	assert.Nil(t, err)
	err = cloud.assertContentUsingSecret(secret)
	assert.Nil(t, err)
	assert.NotEqual(t, secret, cloud.parent.Cookie.Value)

	err = cloud.assertContentUsingSecret(secret)
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "secret does not exist\n"), err.Error())
}

func TestSecretValidation(t *testing.T) {
	if tools.Profile == tools.DOCKER_TEST {
		cloud := getClient(t)
		defer cloud.wipeData()
		assert.Nil(t, cloud.login())
		cloud.setHostValue("localhost")

		_, err := cloud.getSecret()
		assert.Nil(t, err)

		randomSecret := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		err = cloud.assertContentUsingSecret(randomSecret)
		assert.NotNil(t, err)
		assert.Equal(t, utils.GetErrMsg(400, "secret does not exist\n"), err.Error())
	} else {
		t.Skip()
	}
}

func TestAppSearchValidation(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer cloud.wipeData()
	cloud.appToOperateOn = "ab!"
	_, err := cloud.searchForSampleApp()
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(400, "invalid input"), err.Error())
}

func TestRoleVerificationForEndpoints(t *testing.T) {
	adminClient := getClientAndLogin(t)
	defer adminClient.wipeData()
	assert.Nil(t, adminClient.createUser("user", "password"))

	userClient := getClient(t)
	userClient.parent.User = "user"
	userClient.parent.Password = "password"
	assert.Nil(t, userClient.login())
	err := userClient.startApp("123")
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(401, "Unauthorized"), err.Error())

	anonymousClient := getClient(t) // no cookie
	_, err = anonymousClient.getHostValue()
	assert.NotNil(t, err)
	assert.Equal(t, utils.GetErrMsg(401, "cookie required, but none found"), err.Error())
}
