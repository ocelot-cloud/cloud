package cloud

import (
	"fmt"
	"github.com/ocelot-cloud/shared/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"ocelot/backend/tools"
	"testing"
)

func TestGetRequestHostFromHost(t *testing.T) {
	assert.Equal(t, "example.com", getHostFromRequestHost("example.com:8080"))
	assert.Equal(t, "example.com", getHostFromRequestHost("example.com"))
	assert.Equal(t, "", getHostFromRequestHost(""))
}

func TestIsQuerySecretPresent(t *testing.T) {
	sampleSecret := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tests := []struct {
		name        string
		queryParams url.Values
		expected    string
		isPresent   bool
		isValid     bool
	}{
		{"secret present", url.Values{tools.OcelotQuerySecretName: {sampleSecret}}, sampleSecret, true, true},
		{"secret missing", url.Values{}, "", false, true},
		{"secret empty", url.Values{tools.OcelotQuerySecretName: {""}}, "", false, true},
		{"invalid secret value", url.Values{tools.OcelotQuerySecretName: {"invalid"}}, "", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{URL: &url.URL{RawQuery: tc.queryParams.Encode()}}
			secret, ok, isValid := isQuerySecretPresent(r)
			require.Equal(t, tc.expected, secret)
			require.Equal(t, tc.isPresent, ok)
			require.Equal(t, tc.isValid, isValid)
		})
	}
}

func TestBuildTargetURL(t *testing.T) {
	u, err := buildTargetURL("container", "8080", "/path")
	require.NoError(t, err)
	require.Equal(t, "http://container:8080/path", u.String())

	u, err = buildTargetURL("://", "badport", "%%%")
	require.Error(t, err)
}

func TestGetTarget(t *testing.T) {
	appConfigs = map[string]AppConfig{
		"gitea": {Port: 3000, UrlPath: "/some/path2"},
	}
	defer func() { appConfigs = nil }()

	target, err := getTarget("gitea.localhost", "/some/path", "localhost")
	require.NoError(t, err)
	require.Equal(t, "gitea", target.Container)
	require.Equal(t, "3000", target.Port)
	require.Equal(t, "http://gitea:3000/some/path", target.URL.String())

	_, err = getTarget("gitea.localhost", "/some/path", "localhost2")
	assert.NotNil(t, err)
	assert.Equal(t, "internal error", err.Error())
}

func TestCreateProxyRequest(t *testing.T) {
	targetURL, _ := url.Parse("http://container:8080")
	target := Target{
		Container: "container",
		Port:      "8080",
		URL:       targetURL,
	}
	testUrl := fmt.Sprintf("https://originalhost/path?%s=value&other=ok", tools.OcelotQuerySecretName)
	r := httptest.NewRequest(http.MethodGet, testUrl, nil)
	r.Host = "originalhost"
	r.Header.Set(tools.OcelotAuthCookieName, "cookievalue")
	ocelotCookie := &http.Cookie{
		Name:  tools.OcelotAuthCookieName,
		Value: "cookievalue",
	}
	r.AddCookie(ocelotCookie)

	proxy := createProxyRequest(r, target)
	r2 := r.Clone(r.Context())
	proxy.Director(r2)
	assert.Equal(t, "originalhost", r2.Host)
	assert.Equal(t, "http", r2.URL.Scheme)
	assert.Equal(t, "container:8080", r2.URL.Host)
	assert.Equal(t, "other=ok", r2.URL.RawQuery)
	assert.Equal(t, "originalhost", r2.Header.Get("X-Forwarded-Host"))
	assert.Equal(t, "https", r2.Header.Get("X-Forwarded-Proto"))
	assert.Equal(t, "", r2.Header.Get(tools.OcelotAuthCookieName))
	assert.Equal(t, 0, len(r2.Cookies()))
}
