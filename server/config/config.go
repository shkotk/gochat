package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Debug        bool
	LogLevel     string
	PGConnString string
	Port         int
	JWT          JWTConfig
	TLS          TLSConfig
}

type JWTConfig struct {
	Key        string
	Expiration time.Duration
}

type TLSConfig struct {
	CertPath string
	KeyPath  string
}

func Load(pathes ...string) Config {
	for _, path := range pathes {
		godotenv.Load(path)
	}

	envs := loadEnvsMap()

	return Config{
		Debug:        getRequiredString(envs, "DEBUG") == "1",
		LogLevel:     getRequiredString(envs, "LOG_LEVEL"),
		PGConnString: getRequiredString(envs, "PG_CONNECTION_STRING"),
		Port:         getRequiredInt(envs, "PORT"),
		JWT: JWTConfig{
			Key:        getRequiredString(envs, "JWT_KEY"),
			Expiration: getRequiredDuration(envs, "JWT_EXPIRATION"),
		},
		TLS: TLSConfig{
			CertPath: getRequiredString(envs, "TLS_CERT_PATH"),
			KeyPath:  getRequiredString(envs, "TLS_KEY_PATH"),
		},
	}
}

func loadEnvsMap() map[string]string {
	envs := os.Environ()
	envsMap := make(map[string]string, len(envs))
	for _, env := range envs {
		i := strings.IndexRune(env, '=')
		envsMap[env[:i]] = env[i+1:]
	}

	return envsMap
}

func getRequiredDuration(envs map[string]string, key string) time.Duration {
	s := getRequiredString(envs, key)
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf(`can't parse duration from "%s" config value '%s', error: %s`, key, s, err)
	}

	return d
}

func getRequiredInt(envs map[string]string, key string) int {
	s := getRequiredString(envs, key)
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf(`can't parse integer from "%s" config value '%s', error: %s`, key, s, err)
	}

	return i
}

func getRequiredString(envs map[string]string, key string) string {
	value := envs[key]
	if value == "" {
		log.Fatalf(`"%s" config is required, but was empty or missing`, key)
	}

	return value
}
