//go:build fast

package settings

import (
	"github.com/ocelot-cloud/shared/assert"
	"ocelot/backend/apps/common"
	"testing"
)

func TestMain(m *testing.M) {
	common.InitializeDatabase(false, false)
	defer common.WipeWholeDatabase()
	m.Run()
}

func TestKeyStorage(t *testing.T) {
	var key1 ConfigFieldKey = "some-key"
	value1_1 := "some-value"

	_, err := ConfigsRepo.GetValue(key1)
	assert.NotNil(t, err)
	assert.Equal(t, "sql: no rows in result set", err.Error())

	assert.Nil(t, ConfigsRepo.SetConfigField(key1, value1_1))
	outputValue, err := ConfigsRepo.GetValue(key1)
	assert.Nil(t, err)
	assert.Equal(t, value1_1, outputValue)

	value1_2 := "some-value2"
	assert.Nil(t, ConfigsRepo.SetConfigField(key1, value1_2))
	outputValue, err = ConfigsRepo.GetValue(key1)
	assert.Nil(t, err)
	assert.Equal(t, value1_2, outputValue)

	var key2 ConfigFieldKey = "other-key"
	value2 := "very-other-value"
	assert.Nil(t, ConfigsRepo.SetConfigField(key2, value2))

	ConfigsRepo.DeleteKey(key1)
	_, err = ConfigsRepo.GetValue(key1)
	assert.NotNil(t, err)
	assert.Equal(t, "sql: no rows in result set", err.Error())
}
