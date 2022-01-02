package options

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// ServiceOptions is unique to http server / api services
type ServiceOptions struct {
	Limit int  `long:"limit" env:"LIMIT" default:"1000" description:"maximum permitted http connections"`
	SSL   bool `long:"ssl" env:"ENABLE_SSL" description:"enable SSL, default key and crt will be binary name .crt and .key"`
}

// ApplicationOptions defines some default application options present in every utility or server
type ApplicationOptions struct {
	Debug       bool   `short:"d" long:"debug" env:"DEBUG" description:"enable debug logging level"`
	Environment string `short:"e" long:"env" env:"ENVIRONMENT" default:"dev" description:"environment this is running in"`
	Version     bool   `short:"v" long:"version" description:"output version variables"`
}

// Environment loads environment files from a standard configuration place
func Environment(env string) {
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = "dev"
	}
	os.Setenv("ENVIRONMENT", env)
	godotenv.Load("/etc/services/environment/" + env)
	godotenv.Load("/etc/services/" + os.Args[0] + "/environment")
	godotenv.Load(os.Args[0] + "-" + env + ".env")
}

// LogVersion outputs the version build variables
func LogVersion() {
	log.Info().Str("version", Version).Str("build", Build).Msg("version variables")
}
