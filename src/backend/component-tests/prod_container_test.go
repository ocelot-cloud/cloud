//go:build prod_container

package component_tests

import (
	"github.com/ocelot-cloud/shared/assert"
	"net/http"
	"ocelot/backend/tools"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	tools.Profile = tools.PROD
	m.Run()
}

func TestWipeEndpointIsDisabledInProdMode(t *testing.T) {
	cloud := getClient(t)
	resp, err := cloud.parent.DoRequest(tools.WipePath, nil, "")
	assert.Nil(t, err)
	body := string(resp)
	// Getting back HTML code means that there is no wipe endpoint, and the default endpoint is triggered, which provides frontend resources.
	assert.True(t, strings.Contains(body, "<!DOCTYPE html>"))
	assert.True(t, strings.Contains(body, "<html"))
}

func TestProdIsUsingDifferentCookieThanTestCookie(t *testing.T) {
	cloud := getClientAndLogin(t)
	defer deleteUserManually(cloud)
	assert.Equal(t, len(tools.TestCookieValue), len(cloud.parent.Cookie.Value))
	assert.NotEqual(t, tools.TestCookieValue, cloud.parent.Cookie.Value)

	oldCookieValue := cloud.parent.Cookie.Value
	assert.Nil(t, cloud.login())
	assert.NotEqual(t, oldCookieValue, cloud.parent.Cookie.Value)

	assert.Nil(t, cloud.createUser("user", "password"))
	userCloud := getClient(t)
	userCloud.parent.User = "user"
	userCloud.parent.Password = "password"
	assert.Nil(t, userCloud.login())
	assert.Equal(t, len(tools.TestCookieValue), len(userCloud.parent.Cookie.Value))
	assert.NotEqual(t, tools.TestCookieValue, userCloud.parent.Cookie.Value)
}

func deleteUserManually(cloud *CloudClient) {
	users := cloud.getUsers()
	assert.Equal(cloud.t, 2, len(users))

	for _, user := range users {
		if user.Name == "user" {
			assert.Nil(cloud.t, cloud.deleteUser(user.Id))
			break
		}
	}

	users = cloud.getUsers()
	assert.Equal(cloud.t, 1, len(users))
}

func TestAbsenceOfCorsPolicyDisablingHeadersInResponse(t *testing.T) {
	AssertCorsHeaders(t, "", "", "", "")
}

func TestAssertCookieSameSitePolicy(t *testing.T) {
	client := getClientAndLogin(t)
	assert.Equal(t, http.SameSiteStrictMode, client.parent.Cookie.SameSite)
}
