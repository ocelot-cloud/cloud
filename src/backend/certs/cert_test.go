package certs

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/apps/common"
	"testing"
)

func TestCertManagement(t *testing.T) {
	common.InitializeDatabase(false, false)
	defer common.WipeWholeDatabase()

	present, err := isCertPresent()
	assert.Nil(t, err)
	assert.False(t, present)
	_, err = loadCert()
	assert.NotNil(t, err)

	originalCert, err := GenerateUniversalSelfSignedCert()
	assert.Nil(t, err)

	assert.Nil(t, currentCert)
	assert.Nil(t, SaveCert(originalCert))
	present, err = isCertPresent()
	assert.Nil(t, err)
	assert.True(t, present)
	assert.Equal(t, *originalCert, *currentCert)

	loadedCert, err := loadCert()
	assert.Nil(t, err)
	assert.Equal(t, *originalCert, *loadedCert)
}
