package config

import "database/sql"

var (
	DB      *sql.DB
	FlagLog bool
)
