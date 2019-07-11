package dal

import (
	. "github.com/yasnov/goreceipt/config"
	log "github.com/yasnov/goreceipt/logger"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/log/logrusadapter"
	"github.com/jackc/pgx/stdlib"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

func Connect() (*sqlx.DB, error) {
	logger := logrusadapter.NewLogger(log.CreateDBLogger())

	var err error
	/*Try to use PGX connect pool instead*/
	//connPoolConfig := pgx.ConnPoolConfig{
	//	ConnConfig: pgx.ConnConfig{
	//		Host:     "127.0.0.1",
	//		User:     Config.DB.User,
	//		Password: Config.DB.Password,
	//		Database: Config.DB.Name,
	//		Logger:   logger,
	//	},
	//	AfterConnect:   nil,
	//	MaxConnections: 20,
	//	AcquireTimeout: 30 * time.Second,
	//}
	//pool, err := pgx.NewConnPool(connPoolConfig)
	//db := stdlib.OpenDBFromPool(pool)

	db := stdlib.OpenDB(pgx.ConnConfig{
		Host:     "127.0.0.1",
		User:     Config.DB.User,
		Password: Config.DB.Password,
		Database: Config.DB.Name,
		Logger:   logger,
		LogLevel: pgx.LogLevelWarn, // pgx.LogLevelDebug,pgx.LogLevelError,
	})
	return sqlx.NewDb(db, "pgx"), err
}
