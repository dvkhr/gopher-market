package config

import (
	"errors"
	"flag"
	"gopher-market/internal/logging"
	"os"
)

type Config struct {
	Address   string
	DBDSN     string
	Accrual   string
	SecretKey string
}

var (
	ErrAddressEmpty = errors.New("address is an empty string")
	ErrDBDsnEmpty   = errors.New("database_uri is an empty string")
	ErrAccrualEmpty = errors.New("accrual_address is an empty string")
)

func (cfg *Config) check() error {
	var errs []error

	if len(cfg.Address) == 0 {
		errs = append(errs, ErrAddressEmpty)
	} else if len(cfg.DBDSN) == 0 {
		errs = append(errs, ErrDBDsnEmpty)
	} else if len(cfg.Accrual) == 0 {
		errs = append(errs, ErrAccrualEmpty)
	}
	return errors.Join(errs...)
}

func (cfg *Config) ParseFlags() error {
	flag.StringVar(&cfg.Address, "a", "localhost:8080", "Service address and port")
	flag.StringVar(&cfg.DBDSN, "d", "postgres://admin:12345@localhost:5432/loyalty_bonus_system?sslmode=disable", "The database connection")
	flag.StringVar(&cfg.Accrual, "r", "http://localhost:8080", " Address of the accrual system")

	flag.Parse()

	if envVarAddr := os.Getenv("RUN_ADDRESS"); envVarAddr != "" {
		cfg.Address = envVarAddr
	}

	if envVarDB := os.Getenv("DATABASE_URI"); envVarDB != "" {
		cfg.DBDSN = envVarDB
	}

	if envVarAccr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envVarAccr != "" {
		cfg.Accrual = envVarAccr
	}

	if envSecretKey := os.Getenv("JWT_SECRET_KEY"); envSecretKey != "" {
		cfg.SecretKey = envSecretKey
	} else {
		cfg.SecretKey = "default-secret-key-for-development-only"
		logging.Logg.Warn("JWT_SECRET_KEY is not set. Using default key for development only!")

	}
	return cfg.check()
}
