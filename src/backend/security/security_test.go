//go:build fast

package security

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/tools"
	"testing"
)

func TestIsRequestAddressedToAnApp(t *testing.T) {
	assert.False(t, IsRequestAddressedToAnApp("ocelot-cloud.somedomain.org", ""))
	assert.False(t, IsRequestAddressedToAnApp("somedomain.org", ""))
	assert.False(t, IsRequestAddressedToAnApp("some-app.somedomain.org", ""))
	assert.False(t, IsRequestAddressedToAnApp("some-app.somedomain.org", "127.0.0.1"))
	assert.False(t, IsRequestAddressedToAnApp("127.0.0.1", "127.0.0.1"))
	assert.False(t, IsRequestAddressedToAnApp("127.0.0.1", "somedomain.org"))
	assert.False(t, IsRequestAddressedToAnApp("ocelot-cloud.somedomain.org", "somedomain.org"))

	assert.True(t, IsRequestAddressedToAnApp("some-app.somedomain.org", "somedomain.org"))

}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name        string
		requestHost string
		origin      string
		serverHost  string
		errorMsg    string
	}{
		// actually they are redirected to ocelot-cloud
		{"NoHostSetAppRequest", "app.example.com", "app.example.com", "", ""},
		{"OtherHostSetAppRequest", "app.example.com", "app.example.com", "other.com", ""},

		{"ValidOrigin", "app.example.com", "example.com", "example.com", ""},
		{"ValidOcelotOrigin", "app.example.com", "ocelot-cloud.example.com", "example.com", ""},
		{"InvalidAppOrigin", "app.example.com", "other.com", "example.com", crossRequestsToAppsOnlyFromOcelotCloudOriginErrorMessage},

		{"OcelotHostValid", "ocelot-cloud.example.com", "ocelot-cloud.example.com", "example.com", ""},
		{"OcelotHostWithCrossAppOrigin", "ocelot-cloud.example.com", "example.com", "example.com", CrossRequestsToOcelotCloudNotAllowedErrorMessage},
		{"OcelotHostEmptyServerAndOrigin", "ocelot-cloud.example.com", "", "", ""},
		{"OcelotHostEmptyOrigin", "ocelot-cloud.example.com", "", "example.com", ""},
		{"OcelotHostInvalidOrigin", "ocelot-cloud.example.com", "somewhere.else", "example.com", CrossRequestsToOcelotCloudNotAllowedErrorMessage},

		{"PlatformValid", "example.com", "example.com", "example.com", ""},
		{"PlatformNoServerNoOrigin", "example.com", "", "", ""},
		{"PlatformNoOriginServerSet", "example.com", "", "example.com", ""},
		{"PlatformWithOcelotOrigin", "example.com", "ocelot-cloud.example.com", "example.com", CrossRequestsToOcelotCloudNotAllowedErrorMessage},
		{"PlatformWithRandomOrigin", "example.com", "somewhere.else", "example.com", CrossRequestsToOcelotCloudNotAllowedErrorMessage},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := DoesRequestComplyWithOriginPolicy(tt.requestHost, tt.origin, tt.serverHost)
			if tt.errorMsg == "" {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, err.Error(), tt.errorMsg)
			}
		})
	}
}

func TestHasAccess(t *testing.T) {
	tests := []struct {
		name     string
		level    AccessLevelType
		auth     *tools.Authorization
		expected bool
	}{
		{"anonymous1", Anonymous, nil, true},
		{"anonymous2", Anonymous, &tools.Authorization{IsAdmin: false}, true},
		{"anonymous3", Anonymous, &tools.Authorization{IsAdmin: true}, true},

		{"user1", User, nil, false},
		{"user2", User, &tools.Authorization{IsAdmin: false}, true},
		{"user3", User, &tools.Authorization{IsAdmin: true}, true},

		{"admin1", Admin, nil, false},
		{"admin2", Admin, &tools.Authorization{IsAdmin: true}, true},
		{"admin3", Admin, &tools.Authorization{IsAdmin: false}, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, HasAccess(tt.level, tt.auth), tt.expected)
		})
	}
}

func TestIsCrossRequest(t *testing.T) {
	assert.False(t, isCrossOriginRequest("localhost", "localhost"))
	assert.False(t, isCrossOriginRequest("localhost", ""))
	assert.True(t, isCrossOriginRequest("localhost", "localhost2"))
}
