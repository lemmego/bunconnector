package bunconnector

import (
	"testing"

	"github.com/lemmego/api/config"
)

// The scaffold writes no options key for the mysql or pgsql connections, and
// this asserted one unconditionally — so a generated Bun project with
// DB_CONNECTION=mysql panicked at boot.
func TestSQLConfigWithoutAnOptionsKey(t *testing.T) {
	config.Set("sql", config.M{
		"default": "mysql",
		"connections": config.M{
			"mysql": config.M{
				"driver":   "mysql",
				"database": "lemmego",
				"host":     "localhost",
				"port":     3306,
				"user":     "root",
				"password": "",
			},
		},
	})

	cfg := sqlConfig()
	if cfg.Driver != "mysql" || cfg.Database != "lemmego" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.Host != "localhost" || cfg.Port != 3306 {
		t.Errorf("host/port = %s:%d", cfg.Host, cfg.Port)
	}
}

// A connection missing the optional host settings entirely must still resolve.
func TestSQLConfigWithOnlyTheRequiredKeys(t *testing.T) {
	config.Set("sql", config.M{
		"default":     "pgsql",
		"connections": config.M{"pgsql": config.M{"driver": "postgres", "database": "app"}},
	})

	cfg := sqlConfig()
	if cfg.Driver != "postgres" || cfg.Database != "app" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

// bun accepts sqlite3 as an alias and already skips the host lookup for it;
// this pins that behaviour.
func TestSQLConfigTreatsSqlite3AsSqlite(t *testing.T) {
	config.Set("sql", config.M{
		"default":     "sqlite",
		"connections": config.M{"sqlite": config.M{"driver": "sqlite3", "database": "./app.db"}},
	})

	cfg := sqlConfig()
	if cfg.Host != "" || cfg.Port != 0 {
		t.Errorf("sqlite3 was treated as a networked driver: %+v", cfg)
	}
}

// A missing configuration is a mistake worth reporting, and the message should
// name what is missing.
func TestSQLConfigReportsAMissingConnection(t *testing.T) {
	config.Set("sql", config.M{"default": "nowhere", "connections": config.M{}})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("sqlConfig() accepted a missing connection")
		}
		if msg, ok := r.(string); !ok || !contains(msg, "nowhere") {
			t.Errorf("panic message does not name the connection: %v", r)
		}
	}()
	sqlConfig()
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
