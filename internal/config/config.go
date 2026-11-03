// Paket config čita podešavanja iz env promenljivih. Aplikacija ne čita
// nikakav konfiguracioni fajl: sve se zadaje spolja, isto lokalno, u
// kontejneru i u klasteru.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
)

// colorRe prihvata ime CSS boje ili #rgb, #rgba, #rrggbb i #rrggbbaa.
var colorRe = regexp.MustCompile(`^([a-zA-Z]+|#([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8}))$`)

// Config su sva podešavanja aplikacije, već proverena.
type Config struct {
	Port      int
	AppColor  string
	LogLevel  slog.Level
	LogFormat string // "json" ili "text"
	// DatabaseURL je adresa baze Postgres; prazno znači memorijsko skladište.
	DatabaseURL string
}

// Load čita env promenljive. Za neispravnu vrednost vraća grešku koja kaže
// koja promenljiva i zašto, umesto da aplikacija tiho krene sa pogrešnim
// podešavanjem.
func Load() (Config, error) {
	return load(os.Getenv)
}

// load prima funkciju za čitanje promenljivih, da bi testovi mogli da
// proslede svoje vrednosti bez diranja pravog okruženja.
func load(getenv func(string) string) (Config, error) {
	get := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}

	var c Config

	port, err := strconv.Atoi(get("PORT", "8080"))
	if err != nil || port < 1 || port > 65535 {
		return c, fmt.Errorf("PORT must be a number from 1 to 65535, got %q", getenv("PORT"))
	}
	c.Port = port

	// Boja ide u CSS stranice. Dozvoljeno je ime boje (tomato) ili #hex
	// (#f63, #ff6347). Sve ostalo, na primer rgb(...), HTML šablon iz
	// bezbednosnih razloga zameni sa ZgotmplZ, pa bi traka tiho ostala bez boje.
	c.AppColor = get("APP_COLOR", "steelblue")
	if !colorRe.MatchString(c.AppColor) {
		return c, fmt.Errorf("APP_COLOR must be a CSS color name or #rrggbb, got %q", c.AppColor)
	}

	switch lvl := get("LOG_LEVEL", "info"); lvl {
	case "debug":
		c.LogLevel = slog.LevelDebug
	case "info":
		c.LogLevel = slog.LevelInfo
	case "warn":
		c.LogLevel = slog.LevelWarn
	case "error":
		c.LogLevel = slog.LevelError
	default:
		return c, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error, got %q", lvl)
	}

	switch f := get("LOG_FORMAT", "json"); f {
	case "json", "text":
		c.LogFormat = f
	default:
		return c, fmt.Errorf("LOG_FORMAT must be json or text, got %q", f)
	}

	// URL se ne proverava ovde: to radi drajver baze, pri otvaranju pool-a.
	c.DatabaseURL = getenv("DATABASE_URL")

	return c, nil
}
