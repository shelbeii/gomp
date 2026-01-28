package gomp

import (
	"os"

	"gopkg.in/yaml.v3"
)

var config struct {
	Gomp struct {
		EnableSQLPrint    bool `yaml:"enableSqlPrint"`
		AllowGlobalUpdate bool `yaml:"allowGlobalUpdate"`
		AllowGlobalDelete bool `yaml:"allowGlobalDelete"`
	} `yaml:"gomp"`
}

// InitConfig initializes the configuration from a YAML file.
// filePath: absolute or relative path to the yaml configuration file.
func InitConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &config)
}
