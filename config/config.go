package config

import (
	"os"
	"strconv"
	"time"
)

var Config *config

type DatabaseConfig struct {
	Name     string
	User     string
	Password string
}

type NalogConfig struct {
	Login    string
	Password string
}

type config struct {
	DB         DatabaseConfig
	Nalog      NalogConfig
	LogPath    string
	StartTime  string
	UserTestID int
}

// New returns a new config struct
func New() *config {
	return &config{
		DB: DatabaseConfig{
			Name:     getEnv("DB_NAME", ""),
			User:     getEnv("DB_USER", ""),
			Password: getEnv("DB_PASS", ""),
		},
		Nalog: NalogConfig{
			Login:    getEnv("GITHUB_USERNAME", ""),
			Password: getEnv("GITHUB_API_KEY", ""),
		},
		LogPath:    getEnv("LOGPATH", "."),
		StartTime:  time.Now().Format("2006-01-02T15:04:05"),
		UserTestID: getEnvAsInt("TEST_USERID", 1),
	}
}

// Simple helper function to read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

// Simple helper function to read an environment variable into integer or return a default value
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}
