package db

type Config struct {
	Debug                  bool   `yaml:"debug"`
	Driver                 string `yaml:"driver"`
	AccessType             string `yaml:"accessType"`
	DSN                    string `yaml:"dsn"`
	MaxOpenConns           int    `yaml:"maxOpenConns"`
	MaxIdleConns           int    `yaml:"maxIdleConns"`
	ConnMaxLifetimeSeconds int    `yaml:"connMaxLifetimeSeconds"`
	ConnMaxIdleSeconds     int    `yaml:"connMaxIdleSeconds"`

	SQLPath string `yaml:"sqlPath"`
	Migrate bool   `yaml:"migrate"`
}

const (
	MATRIXHUB_DSN_ENV = "MATRIXHUB_DATABASE_DSN"
)
