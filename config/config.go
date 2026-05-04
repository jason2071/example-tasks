package config

import (
	"bytes"
	_ "embed"
	"strings"

	"example-tasks/model"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

//go:embed config.yaml
var configFile []byte

func LoadConfig() (*model.AppConfig, error) {
	// โหลด .env เฉพาะ local
	_ = gotenv.Load()

	v := viper.New()

	v.SetConfigType("yaml")

	// ENV override
	v.AutomaticEnv()

	// map ENV → config key
	v.SetEnvKeyReplacer(
		strings.NewReplacer(".", "__", "-", "_"),
	)

	// อ่าน config.yaml
	if err := v.ReadConfig(bytes.NewBuffer(configFile)); err != nil {
		return nil, err
	}

	cfg := &model.AppConfig{}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
