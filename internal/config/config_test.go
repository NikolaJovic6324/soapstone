package config

import (
	"log/slog"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	c, err := load(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Port: 8080, AppColor: "steelblue", LogLevel: slog.LevelInfo, LogFormat: "json"}
	if c != want {
		t.Errorf("load() = %+v, want %+v", c, want)
	}
}

func TestLoadValues(t *testing.T) {
	c, err := load(env(map[string]string{
		"PORT": "9090", "APP_COLOR": "tomato", "LOG_LEVEL": "debug", "LOG_FORMAT": "text",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Port: 9090, AppColor: "tomato", LogLevel: slog.LevelDebug, LogFormat: "text"}
	if c != want {
		t.Errorf("load() = %+v, want %+v", c, want)
	}
}

func TestLoadInvalid(t *testing.T) {
	for _, m := range []map[string]string{
		{"PORT": "eighty"},
		{"PORT": "0"},
		{"PORT": "70000"},
		{"LOG_LEVEL": "loud"},
		{"LOG_FORMAT": "xml"},
		{"APP_COLOR": "rgb(255,99,71)"},
		{"APP_COLOR": "tomato red"},
		{"APP_COLOR": "#ff634"},
		{"APP_COLOR": "red;}</style>"},
	} {
		if _, err := load(env(m)); err == nil {
			t.Errorf("load(%v) returned no error", m)
		}
	}
}

func TestLoadColors(t *testing.T) {
	for _, color := range []string{"tomato", "SteelBlue", "#f63", "#f63a", "#ff6347", "#ff6347cc"} {
		c, err := load(env(map[string]string{"APP_COLOR": color}))
		if err != nil || c.AppColor != color {
			t.Errorf("APP_COLOR=%q: %+v, %v", color, c, err)
		}
	}
}

func TestLoadDatabaseURL(t *testing.T) {
	url := "postgres://soapstone:tajna@db:5432/soapstone?sslmode=disable"
	c, err := load(env(map[string]string{"DATABASE_URL": url}))
	if err != nil {
		t.Fatal(err)
	}
	if c.DatabaseURL != url {
		t.Errorf("DatabaseURL = %q, want %q", c.DatabaseURL, url)
	}
}
