package db

import (
	"testing"
)

func TestGetAllDiscsSqlQueryCompiles(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Errorf("failed to open in-memory db: %+v", err.Error())
	}
	// db is empty at this point, so this purely tests if the sql
	// query is syntactically correct.
	r, err := GetAllDiscs(db)
	if err != nil {
		t.Errorf("failed to construct sql query: %+v", err.Error())
	}
	count := 0
	want := 0
	for r.Next() {
		count++
	}
	if count != want {
		t.Errorf("got %v, want %v rows", count, want)
	}
}

func TestGetAllDsicsSingleEntry(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Errorf("failed to open in-memory db: %+v", err.Error())
	}
	// Insert a disc fingerprint
	disc := DiscFingerprint{}
	if err := db.Create(&disc).Error; err != nil {
		t.Errorf("failed to insert disc record: %+v", err.Error())
	}
	// Insert a session
	session := Session{
		DiscFingerprintID: &disc.ID,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Errorf("failed to insert session record: %+v", err.Error())	
	}
	// Insert a log
	if err := db.Model(&session).Association("RawLog").Append(&MakeMkvLog{
		Args: []string{"info"},
	}); err != nil {
		t.Errorf("failed to insert log record: %+v", err)
	}
	// Now, check that we got exactly 1 row
	r, err := GetAllDiscs(db)
	if err != nil {
		t.Errorf("failed to construct sql query: %+v", err.Error())
	}

	count := 0
	want := 1
	for r.Next() {
		count++
	}
	if count != want {
		t.Errorf("got %v, want %v rows", count, want)
	}	
}

func TestGetAllDsicsMultipleEntry(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Errorf("failed to open in-memory db: %+v", err.Error())
	}
	// Insert a disc fingerprint
	disc := DiscFingerprint{}
	if err := db.Create(&disc).Error; err != nil {
		t.Errorf("failed to insert disc record: %+v", err.Error())
	}
	// Insert a session
	session := Session{
		DiscFingerprintID: &disc.ID,
	}
	if err := db.Create(&session).Error; err != nil {
		t.Errorf("failed to insert session record: %+v", err.Error())	
	}
	// Insert a log
	if err := db.Model(&session).Association("RawLog").Append(&MakeMkvLog{
		Args: []string{"totallynotinfo"},
	}); err != nil {
		t.Errorf("failed to insert log record: %+v", err)
	}
	if err := db.Model(&session).Association("RawLog").Append(&MakeMkvLog{
		Args: []string{"info"},
	}); err != nil {
		t.Errorf("failed to insert log record: %+v", err)
	}
	// Now, check that we got exactly 1 row
	r, err := GetAllDiscs(db)
	if err != nil {
		t.Errorf("failed to construct sql query: %+v", err.Error())
	}

	count := 0
	want := 1
	for r.Next() {
		count++
	}
	if count != want {
		t.Errorf("got %v, want %v rows", count, want)
	}	
}
