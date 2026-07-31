package auth

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const defaultDevJWTSecret = "camper-vane-dev-secret-key-32bytes!"

var (
	cookieSecure bool
	appEnv       string
)

// InitFromEnv configures JWT signing and cookie security from environment.
// Production (APP_ENV=production) rejects missing/default JWT secrets and enables Secure cookies by default.
func InitFromEnv() error {
	appEnv = strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if appEnv == "" {
		appEnv = "development"
	}

	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if appEnv == "production" {
		if secret == "" || secret == defaultDevJWTSecret {
			return fmt.Errorf("JWT_SECRET must be set to a non-default value when APP_ENV=production")
		}
		SetJWTSecret(secret)
	} else if secret != "" {
		SetJWTSecret(secret)
	} else {
		SetJWTSecret(defaultDevJWTSecret)
		log.Printf("auth: using default JWT secret (set JWT_SECRET for non-dev environments)")
	}

	secureFlag := strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE")))
	switch secureFlag {
	case "true", "1", "yes":
		cookieSecure = true
	case "false", "0", "no":
		cookieSecure = false
	default:
		cookieSecure = appEnv == "production"
	}

	log.Printf("auth: env=%s cookie_secure=%v", appEnv, cookieSecure)
	return nil
}

func CookieSecure() bool {
	return cookieSecure
}

func AppEnv() string {
	return appEnv
}

func IsProduction() bool {
	return appEnv == "production"
}
