package common

import (
	"github.com/songquanpeng/one-api/common/env"
)

var (
	UsingSQLite     = false
	UsingPostgreSQL = false
	UsingMySQL      = false
)

var (
	SQLitePath        = "one-api.db"
	SQLiteBusyTimeout = env.Int("SQLITE_BUSY_TIMEOUT", 3000)
)
