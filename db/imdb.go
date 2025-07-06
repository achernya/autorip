package db

import (
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Title struct {
	gorm.Model
	TConst string `gorm:"index"`
	TitleType string
	PrimaryTitle string
	OriginalTitle string
	IsAdult bool
	StartYear int
	EndYear *int `json:",omitempty"`
	RuntimeMinutes int
	Genres datatypes.JSONSlice[string]
	// Only populated for tvSeries
	Episodes []*Title `gorm:"many2many:episodes;" json:",omitempty"`
	// Only populated for tvEpisode
	SeasonNumber *int `json:",omitempty"`
	EpisodeNumber *int `json:",omitempty"`
}

func OpenImdb(dsn string)  (*gorm.DB, error) {	
	db, err := gorm.Open(sqlite.Open(dsn))
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&Title{}); err != nil {
		return nil, err
	}
	return db, nil
}

