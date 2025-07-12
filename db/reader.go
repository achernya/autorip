package db

import (
	"bytes"
	"io"
	"database/sql"

	"gorm.io/gorm"
)

type logReader struct {
	db         *gorm.DB
	rows       *sql.Rows
	currReader io.Reader
}

func (r *logReader) Read(p []byte) (n int, err error) {
	if r.currReader == nil {
		// No reader right now, try to fetch the next row.
		if !r.rows.Next() {
			return 0, io.EOF
		}
		var id uint
		var b []byte
		if err := r.rows.Scan(&id, &b); err != nil {
			return 0, io.EOF
		}
		r.currReader = bytes.NewReader(append(b, '\n'))
	}
	n, err = r.currReader.Read(p)
	if err == io.EOF {
		r.currReader = nil
		err = nil
	}
	return
}

func NewLogReader(db *gorm.DB, logid uint) (io.Reader, error) {
	result := &logReader{
		db:  db,
	}
	rows, err := db.Raw("SELECT id,entry FROM make_mkv_log_entries WHERE make_mkv_log_id = ? ORDER BY id ASC", logid).Rows()
	if err != nil {
		return nil, err
	}
	result.rows = rows
	return result, nil
}
