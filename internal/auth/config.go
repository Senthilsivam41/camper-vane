package auth

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

const defaultDevJWTSecret = "camper-vane-dev-secret-key-32bytes!"

var (
	cookieSecure   bool
	cookieSameSite http.SameSite
	appEnv         string
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

	sameSiteFlag := strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SAMESITE")))
	switch sameSiteFlag {
	case "strict":
		cookieSameSite = http.SameSiteStrictMode
	case "none":
		cookieSameSite = http.SameSiteNoneMode
		// Browsers require Secure when SameSite=None.
		if !cookieSecure {
			log.Printf("auth: COOKIE_SAMESITE=none forces COOKIE_SECURE=true")
			cookieSecure = true
		}
	case "lax", "":
		cookieSameSite = http.SameSiteLaxMode
	default:
		return fmt.Errorf("COOKIE_SAMESITE must be lax, strict, or none (got %q)", sameSiteFlag)
	}

	log.Printf("auth: env=%s cookie_secure=%v cookie_samesite=%s", appEnv, cookieSecure, sameSiteName(cookieSameSite))
	return nil
}

func CookieSecure() bool {
	return cookieSecure
}

func CookieSameSite() http.SameSite {
	if cookieSameSite == 0 {
		return http.SameSiteLaxMode
	}
	return cookieSameSite
}

func AppEnv() string {
	return appEnv
}

func IsProduction() bool {
	return appEnv == "production"
}

func sameSiteName(m http.SameSite) string {
	switch m {
	case http.SameSiteStrictMode:
		return "strict"
	case http.SameSiteNoneMode:
		return "none"
	default:
		return "lax"
	}
}
