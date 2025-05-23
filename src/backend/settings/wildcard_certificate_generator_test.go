package settings

import (
	"fmt"
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/tools"
	"testing"
)

// The default setting is 'true' because this test requires a developer to intervene manually. If it were to be executed in GitHub Actions, it would crash the CI pipeline. For testing and development purposes, however, it should be temporarily set to 'false'.
const skipManualTest = true

func TestCertCreation(t *testing.T) {
	if skipManualTest {
		t.Skip("skipping manual cert creation test")
		return
	}
	data, err := crateCertificateViaLetsEncryptDns01Challenge("cert-test.ocelot-cloud.org", "", tools.FAKE_LETSENCRYPT_CERTIFICATE)
	assert.Nil(t, err)
	printDnsSetupInstructions(data.Name, data.BaseKeyAuth, data.WildcardKeyAuth)
}

func printDnsSetupInstructions(host, baseKeyAuth, wildcardKeyAuth string) {
	fmt.Printf("\nCreate a DNS TXT record for '%s%s' with:\n%s\n%s\n\n", acmeChallengePrefix, host, baseKeyAuth, wildcardKeyAuth)
}

func TestGetDnsRecordStub(t *testing.T) {
	record := getDnsRecordStub("sample.com")
	assert.Equal(t, acmeChallengePrefix+"sample.com", record.Name)
	assert.Equal(t, sampleBaseKeyAuth, record.BaseKeyAuth)
	assert.Equal(t, sampleWildcardKeyAuth, record.WildcardKeyAuth)
}
