package test

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type TestConfig struct {
	DBConnString string
}

func LoadConfig(path string) TestConfig {
	godotenv.Load(path)

	return TestConfig{
		DBConnString: readRequiredConfig("TEST_PG_CONNECTION_STRING"),
	}
}

func readRequiredConfig(configKey string) string {
	configValue := os.Getenv(configKey)
	if configValue == "" {
		log.Fatalf("'%v' config is required, but was empty or missing", configKey)
	}

	return configValue
}
