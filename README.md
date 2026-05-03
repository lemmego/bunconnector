### Lemmego Bun Connector

The bunconnector plugin for the Lemmego framework. Provides database connectivity using [uptrace/bun](https://github.com/uptrace/bun) with optional GPA integration.

#### Usage

```go
// With GPA
&bunconnector.Provider{UseGPA: true}

// Without GPA (raw *bun.DB)
&bunconnector.Provider{UseGPA: false}
```

#### CLI Commands

- `bun:model` — Generate a Bun model with `bun:` struct tags
- `bun:repo` — Generate a repository with GPA integration

#### Supported Drivers

- PostgreSQL (`postgres`, `postgresql`)
- MySQL (`mysql`)
- SQLite (`sqlite`, `sqlite3`)
