package model

type AppConfig struct {
	AppInfo  AppInfo        `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
}

type AppInfo struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Description string `mapstructure:"description"`
	Environment string `mapstructure:"environment"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"dbname"`
	Sslmode  string `mapstructure:"sslmode"`
}
