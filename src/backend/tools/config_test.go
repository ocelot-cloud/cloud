//go:build fast

package tools

import (
	"github.com/ocelot-cloud/shared/assert"
	"testing"
)

func TestProfileStructs(t *testing.T) {
	assert.Equal(t, "PROD", PROD.String())
	assert.Equal(t, "NATIVE", NATIVE.String())
	assert.Equal(t, "DOCKER_TEST", DOCKER_TEST.String())
}

func TestGetConfig(t *testing.T) {
	nativeConfig := getGlobalConfigBasedOnProfile(NATIVE)
	assert.Equal(t, false, nativeConfig.IsGuiEnabled)
	assert.Equal(t, true, nativeConfig.AreCrossOriginRequestsAllowed)
	assert.Equal(t, true, nativeConfig.OpenDataWipeEndpoint)
	assert.Equal(t, false, nativeConfig.UseRealAppStoreClient)
	assert.Equal(t, false, nativeConfig.IsUsingDockerNetwork)
	assert.Equal(t, false, nativeConfig.UseProductionDatabaseContainer)
	assert.Equal(t, false, nativeConfig.IsMaintenanceAgentEnabled)
	assert.Equal(t, STUB_CERTIFICATE, nativeConfig.CertificateDnsChallengeClient)
	assert.True(t, nativeConfig.UseMockedSshClient)

	dockerTestConfig := getGlobalConfigBasedOnProfile(DOCKER_TEST)
	assert.Equal(t, true, dockerTestConfig.IsGuiEnabled)
	assert.Equal(t, false, dockerTestConfig.AreCrossOriginRequestsAllowed)
	assert.Equal(t, true, dockerTestConfig.OpenDataWipeEndpoint)
	assert.Equal(t, false, dockerTestConfig.UseRealAppStoreClient)
	assert.Equal(t, true, dockerTestConfig.IsUsingDockerNetwork)
	assert.Equal(t, false, dockerTestConfig.UseProductionDatabaseContainer)
	assert.Equal(t, false, dockerTestConfig.IsMaintenanceAgentEnabled)
	assert.Equal(t, FAKE_LETSENCRYPT_CERTIFICATE, dockerTestConfig.CertificateDnsChallengeClient)
	assert.True(t, dockerTestConfig.UseMockedSshClient)

	prodConfig := getGlobalConfigBasedOnProfile(PROD)
	assert.Equal(t, true, prodConfig.IsGuiEnabled)
	assert.Equal(t, false, prodConfig.AreCrossOriginRequestsAllowed)
	assert.Equal(t, false, prodConfig.OpenDataWipeEndpoint)
	assert.Equal(t, true, prodConfig.UseRealAppStoreClient)
	assert.Equal(t, true, prodConfig.IsUsingDockerNetwork)
	assert.Equal(t, true, prodConfig.UseProductionDatabaseContainer)
	assert.Equal(t, true, prodConfig.IsMaintenanceAgentEnabled)
	assert.Equal(t, PRODUCTION_LETSENCRYPT_CERTIFICATE, prodConfig.CertificateDnsChallengeClient)
	assert.False(t, prodConfig.UseMockedSshClient)
}
