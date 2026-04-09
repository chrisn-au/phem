package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr     string
	DBHost       string
	DBPort       int
	DBUser       string
	DBPassword   string
	DBName       string
	SeedOnEmpty  bool
	SiteLat      float64
	SiteLon      float64
	SiteTZ       string
}

func FromEnv() Config {
	return Config{
		HTTPAddr:    getenv("PHEM_HTTP_ADDR", ":8080"),
		DBHost:      getenv("PHEM_DB_HOST", "localhost"),
		DBPort:      getenvInt("PHEM_DB_PORT", 5432),
		DBUser:      getenv("PHEM_DB_USER", "phem"),
		DBPassword:  getenv("PHEM_DB_PASSWORD", "phem"),
		DBName:      getenv("PHEM_DB_NAME", "phem"),
		SeedOnEmpty: getenv("PHEM_SEED_ON_EMPTY", "true") == "true",
		SiteLat:     getenvFloat("PHEM_SITE_LAT", -33.8688),
		SiteLon:     getenvFloat("PHEM_SITE_LON", 151.2093),
		SiteTZ:      getenv("PHEM_SITE_TZ", "Australia/Sydney"),
	}
}

func (c Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvFloat(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
