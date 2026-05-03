package bunconnector

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	"github.com/lemmego/gpa"
	"github.com/lemmego/gpabun"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
)

type Provider struct {
	UseGPA    bool
	config    gpa.Config
	appConfig config.Configuration
	sqlDB     *sql.DB
}

func (b *Provider) WithGPAConfig(config gpa.Config) *Provider {
	b.config = config
	return b
}

func (b *Provider) AddCommands() []app.Command {
	return []app.Command{
		genBunModelCmd,
		genBunRepoCmd,
	}
}

func (b *Provider) GetSQLDb() *sql.DB {
	return b.sqlDB
}

func (b *Provider) Provide(a app.App) error {
	b.appConfig = a.Config()
	dbConfig := sqlConfig()
	if b.config.Host != "" {
		dbConfig = b.config
	}

	if b.UseGPA {
		provider, err := gpabun.NewProvider(dbConfig)
		if err != nil {
			panic(err)
		}
		b.sqlDB = provider.DB().(*sql.DB)
		gpa.RegisterDefault(provider)
		a.AddService(provider)
	} else {
		db, err := NewBunConnection(dbConfig)
		if err != nil {
			panic(err)
		}
		b.sqlDB = db.DB
		a.AddService(db)
	}

	return nil
}

func (b *Provider) Shutdown(ctx context.Context) error {
	if b.UseGPA {
		gpa.Registry().RemoveAll()
	}
	if b.sqlDB != nil {
		return b.sqlDB.Close()
	}
	return nil
}

func NewBunConnection(config gpa.Config) (*bun.DB, error) {
	var sqlDB *sql.DB
	var err error

	switch strings.ToLower(config.Driver) {
	case "postgres", "postgresql":
		sqlDB = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(buildPostgresDSN(config))))
	case "mysql":
		sqlDB, err = openMySQL(config)
	case "sqlite", "sqlite3":
		sqlDB, err = openSQLite(config)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", config.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	var bunDB *bun.DB
	switch strings.ToLower(config.Driver) {
	case "postgres", "postgresql":
		bunDB = bun.NewDB(sqlDB, pgdialect.New())
	case "mysql":
		bunDB = bun.NewDB(sqlDB, mysqldialect.New())
	case "sqlite", "sqlite3":
		bunDB = bun.NewDB(sqlDB, sqlitedialect.New())
	}

	return bunDB, nil
}

func openMySQL(config gpa.Config) (*sql.DB, error) {
	if config.ConnectionURL != "" {
		return sql.Open("mysql", config.ConnectionURL)
	}
	mysqlConfig := mysql.Config{
		User:   config.Username,
		Passwd: config.Password,
		Net:    "tcp",
		Addr:   fmt.Sprintf("%s:%d", config.Host, config.Port),
		DBName: config.Database,
	}
	return sql.Open("mysql", mysqlConfig.FormatDSN())
}

func openSQLite(config gpa.Config) (*sql.DB, error) {
	if config.ConnectionURL != "" {
		return sql.Open(sqliteshim.ShimName, config.ConnectionURL)
	}
	return sql.Open(sqliteshim.ShimName, config.Database)
}

func sqlConfig(connName ...string) gpa.Config {
	name := "default"
	if len(connName) > 0 && connName[0] != "" {
		name = connName[0]
	}

	defaultConnection := config.Get(fmt.Sprintf("sql.%s", name))
	connection := config.Get(fmt.Sprintf("sql.connections.%s", defaultConnection)).(config.M)
	driver := connection.String("driver")
	database := connection.String("database")

	if database == "" || driver == "" {
		panic("database: database and driver must be present")
	}

	dbConfig := gpa.Config{
		Driver:   driver,
		Database: database,
	}

	if driver != "sqlite" && driver != "sqlite3" {
		dbConfig.Host = config.Get(fmt.Sprintf("sql.connections.%s.host", defaultConnection)).(string)
		dbConfig.Port = config.Get(fmt.Sprintf("sql.connections.%s.port", defaultConnection)).(int)
		dbConfig.Username = config.Get(fmt.Sprintf("sql.connections.%s.user", defaultConnection)).(string)
		dbConfig.Password = config.Get(fmt.Sprintf("sql.connections.%s.password", defaultConnection)).(string)
		dbConfig.Options = config.Get(fmt.Sprintf("sql.connections.%s.options", defaultConnection)).(config.M)
	}

	return dbConfig
}

// =====================================
// Helper Functions
// =====================================

func buildPostgresDSN(config gpa.Config) string {
	if config.ConnectionURL != "" {
		return config.ConnectionURL
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	if config.SSL.Enabled {
		dsn = strings.Replace(dsn, "sslmode=disable", "sslmode="+config.SSL.Mode, 1)
		if config.SSL.CertFile != "" {
			dsn += "&sslcert=" + config.SSL.CertFile
		}
		if config.SSL.KeyFile != "" {
			dsn += "&sslkey=" + config.SSL.KeyFile
		}
		if config.SSL.CAFile != "" {
			dsn += "&sslrootcert=" + config.SSL.CAFile
		}
	}

	return dsn
}

func buildMySQLDSN(config gpa.Config) string {
	if config.ConnectionURL != "" {
		return config.ConnectionURL
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Username, config.Password, config.Host, config.Port, config.Database)

	if config.SSL.Enabled {
		dsn += "&tls=" + config.SSL.Mode
	}

	return dsn
}

func SupportedDrivers() []string {
	return []string{"postgres", "postgresql", "mysql", "sqlite", "sqlite3"}
}

func Get(a app.App) *gpabun.Provider {
	return app.Get[*gpabun.Provider](a)
}
