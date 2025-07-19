package db

import (
	"bufio"
	"reflect"
	"testing"
)

func TestEmptyDB(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Errorf("failed to open in-memory db: %+v", err.Error())
	}
	// db is empty at this point, so this should produce a valid
	// reader that selects nothing.
	r, err := NewLogReader(db, 0)
	if err != nil {
		t.Errorf("failed to construct log reader: %+v", err.Error())
	}
	s := bufio.NewScanner(r)
	if s.Scan() {
		t.Errorf("scanner unexpectedly produced tokens")
	}

}

func TestSingleEntry(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Errorf("failed to open in-memory db: %+v", err.Error())
	}
	// Insert a log entry
	log := &MakeMkvLog{}
	if err := db.Create(log).Error; err != nil {
		t.Errorf("failed to insert log record: %+v", err.Error())
	}
	want := "a log line"
	if err := db.Model(&log).Association("Entry").Append(&MakeMkvLogEntry{
		Entry: want,
	}); err != nil {
		t.Errorf("failed to insert log line: %+v", err.Error())
	}
	r, err := NewLogReader(db, log.ID)
	if err != nil {
		t.Errorf("failed to construct log reader: %+v", err.Error())
	}
	s := bufio.NewScanner(r)
	if !s.Scan() {
		t.Errorf("expected a valid line, but received none")
	}
	if s.Text() != want {
		t.Errorf("got %+q, want %+q", s.Text(), want)
	}
	if s.Scan() {
		t.Errorf("scanner unexpectedly produced tokens")
	}
}
func TestMultipleEntry(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Errorf("failed to open in-memory db: %+v", err.Error())
	}
	// Insert all the log entries
	log := &MakeMkvLog{}
	if err := db.Create(log).Error; err != nil {
		t.Errorf("failed to insert log record: %+v", err.Error())
	}
	want := []string{
		"a log line",
		"another log line",
	}
	for _, line := range want {
		if err := db.Model(&log).Association("Entry").Append(&MakeMkvLogEntry{
			Entry: line,
		}); err != nil {
			t.Errorf("failed to insert log line: %+v", err.Error())
		}
	}
	r, err := NewLogReader(db, log.ID)
	if err != nil {
		t.Errorf("failed to construct log reader: %+v", err.Error())
	}
	s := bufio.NewScanner(r)
	got := []string{}
	for s.Scan() {
		got = append(got, s.Text())
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
