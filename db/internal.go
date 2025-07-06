package db

import (
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Session struct {
	gorm.Model
	RawLog            []MakeMkvLog
	DiscFingerprintID *uint
}

type MakeMkvLog struct {
	gorm.Model
	SessionID uint
	Args      datatypes.JSONSlice[string]
	Entry     []MakeMkvLogEntry
}

type MakeMkvLogEntry struct {
	gorm.Model
	MakeMkvLogID uint
	Entry        string
}

type DiscFingerprint struct {
	gorm.Model
	Fingerprint []byte `gorm:"uniqueIndex"`
	Name        string
	VolumeName  string
}

func OpenDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn))
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&Session{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&MakeMkvLog{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&MakeMkvLogEntry{}); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&DiscFingerprint{}); err != nil {
		return nil, err
	}
	return db, nil
}
