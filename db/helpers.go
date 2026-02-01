package db

import (
	_ "embed"
	"database/sql"

	"gorm.io/gorm"
)

//go:embed queries/disc_and_log.sql
var discAndLogSql string

func GetAllDiscs(db *gorm.DB) (*sql.Rows, error) {
	return db.Raw(discAndLogSql).Rows()
}
