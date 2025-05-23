package setup

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetHostFromOriginHeaderValue(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedHost string
		expectErr    bool
	}{
		{"empty", "", "", false},
		{"null", "null", "", false},
		{"valid", "https://example.com", "example.com", false},
		{"valid", "http://example.com", "example.com", false},
		{"invalid", "example.com", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host, err := getHostFromOriginHeaderValue(tc.input)
			if tc.expectErr {
				require.Error(t, err)
				require.Empty(t, host)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedHost, host)
			}
		})
	}
}
