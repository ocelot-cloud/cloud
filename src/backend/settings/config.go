package settings

import (
	"ocelot/backend/apps/common"
	"ocelot/backend/tools"
)

var (
	Logger      = tools.Logger
	ConfigsRepo = &ConfigsRepository{}
)

type ConfigsRepository struct{}

type ConfigFieldKey string

func (c *ConfigsRepository) SetConfigField(configFieldName ConfigFieldKey, value string) error {
	_, err := common.DB.Exec("INSERT INTO configs (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value", configFieldName, value)
	if err != nil {
		Logger.Error("Failed to set configFieldName %s: %v", configFieldName, err)
		return err
	}
	return nil
}

func (c *ConfigsRepository) GetValue(key ConfigFieldKey) (string, error) {
	var value string
	err := common.DB.QueryRow("SELECT value FROM configs WHERE key = $1", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (c *ConfigsRepository) DeleteKey(key ConfigFieldKey) {
	_, err := common.DB.Exec("DELETE FROM configs WHERE key = $1", key)
	if err != nil {
		Logger.Warn("Failed to delete key %s: %v", key, err)
	}
}
