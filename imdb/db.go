package imdb

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"dario.cat/mergo"
	"github.com/achernya/autorip/db"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	batchSize = 100000
)

func key(keys ...string) []byte {
	return []byte(strings.Join(keys, ","))
}

func decode(b []byte) (*db.Title, error) {
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)
	entry := &db.Title{}
	if err := dec.Decode(entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// lookup finds the given key in the given leveldb database and
// returns it as a decoded Title. No modification to the returned
// object will be performed (i.e., TConst is not filled in).
func lookup(ldb *leveldb.DB, title ...string) (*db.Title, error) {
	b, err := ldb.Get(key(title...), nil)
	if err != nil {
		return nil, err
	}
	entry, err := decode(b)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func findTitle(ldb *leveldb.DB, title string) (*db.Title, error) {
	it := ldb.NewIterator(&util.Range{key(title), nil}, nil)
	if !it.First() {
		return nil, fmt.Errorf("could not find key %+v", title)
	}
	var entry *db.Title = nil
	var err error
	for {
		currKey := strings.Split(string(it.Key()), ",")
		if currKey[0] != title {
			break
		}
		switch len(currKey) {
		case 1:
			if entry != nil {
				return nil, fmt.Errorf("multiple top-level records found for key %+v", title)
			}
			entry, err = decode(it.Value())
			if err != nil {
				return nil, err
			}
			entry.TConst = currKey[0]
			entry.Episodes = make([]*db.Title, 0)
		case 2:
			if entry == nil {
				return nil, fmt.Errorf("encountered subrecord before parent record for key %+v", title)
			}
			subentry, err := decode(it.Value())
			if err != nil {
				return nil, err
			}
			subentry.TConst = currKey[1]
			entry.Episodes = append(entry.Episodes, subentry)
		default:
			return nil, fmt.Errorf("got unexpected %d keys for title %+v", len(currKey), title)
		}
		if !it.Next() {
			break
		}
	}
	// Double-check that some data was found.
	if entry == nil {
		return nil, fmt.Errorf("could not find key %+v", title)
	}

	// Fill in per-episode data
	for _, episode := range entry.Episodes {
		subentry, err := lookup(ldb, episode.TConst)
		if err != nil {
			return nil, err
		}
		mergo.Merge(episode, subentry, mergo.WithoutDereference)
	}
	return entry, nil
}

func loadTitles(ldb *leveldb.DB, dir string) error {
	fgz, err := os.Open(basics)
	if err != nil {
		return err
	}
	f, err := gzip.NewReader(fgz)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	// Consume the header
	scanner.Scan()

	tx, err := ldb.OpenTransaction()
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	count := 0
	batch := leveldb.Batch{}

	for scanner.Scan() {
		line := scanner.Text()
		record := strings.Split(line, "\t")
		title := db.Title{
			//TConst:        record[0],
			TitleType:     record[1],
			PrimaryTitle:  record[2],
			OriginalTitle: record[3],
			Genres:        strings.Split(record[8], ","),
		}
		if b, err := strconv.ParseBool(record[4]); err == nil {
			title.IsAdult = b
		}
		if year, err := strconv.Atoi(record[5]); err == nil {
			title.StartYear = year
		}
		if year, err := strconv.Atoi(record[6]); err == nil {
			title.EndYear = &year
		}
		if runtime, err := strconv.Atoi(record[7]); err == nil {
			title.RuntimeMinutes = runtime
		}
		// Drop any old data before encoding the struct
		buf.Reset()
		enc := gob.NewEncoder(buf)
		if err := enc.Encode(title); err != nil {
			return err
		}
		batch.Put(key(record[0]), buf.Bytes())
		count++
		if count == batchSize {
			if err := tx.Write(&batch, nil); err != nil {
				return err
			}
			batch = leveldb.Batch{}
			count = 0
		}
	}
	if err := tx.Write(&batch, nil); err != nil {
		return err
	}
	return tx.Commit()
}

func loadEpisodes(ldb *leveldb.DB, dir string) error {
	fgz, err := os.Open(episodes)
	if err != nil {
		return err
	}
	f, err := gzip.NewReader(fgz)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	// Consume the header line
	scanner.Scan()

	tx, err := ldb.OpenTransaction()
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	count := 0
	batch := leveldb.Batch{}

	for scanner.Scan() {
		line := scanner.Text()
		record := strings.Split(line, "\t")
		tConst := record[0]
		parentTConst := record[1]
		// First, add the metadata to the episode
		update := db.Title{}
		seasonNumber, err := strconv.Atoi(record[2])
		if err == nil {
			update.SeasonNumber = &seasonNumber
		}
		episodeNumber, err := strconv.Atoi(record[3])
		if err == nil {
			update.EpisodeNumber = &episodeNumber
		}
		// Drop any old data before encoding the struct
		buf.Reset()
		enc := gob.NewEncoder(buf)
		if err := enc.Encode(update); err != nil {
			return err
		}
		batch.Put(key(parentTConst, tConst), buf.Bytes())
		count++
		if count == batchSize {
			if err := tx.Write(&batch, nil); err != nil {
				return err
			}
			batch = leveldb.Batch{}
			count = 0
		}
	}
	if err := tx.Write(&batch, nil); err != nil {
		return err
	}
	return tx.Commit()
}

func MakeIndex(dir string) error {
	ldb, err := leveldb.OpenFile("imdb.leveldb", &opt.Options{
		// Ideally we'd use zstd here, but it's not available.
		Compression: opt.SnappyCompression,
		// We expect to create the database here.
		ErrorIfExist:   true,
		ErrorIfMissing: false,
	})
	if err != nil {
		return err
	}
	defer ldb.Close()

	// Load titles
	log.Println("Loading titles")
	if err := loadTitles(ldb, dir); err != nil {
		return err
	}

	// Load episodes
	log.Println("Loading episodes")
	if err := loadEpisodes(ldb, dir); err != nil {
		return err
	}

	log.Println("Compacting")
	if err := ldb.CompactRange(util.Range{nil, nil}); err != nil {
		return err
	}
	log.Println("Done")
	return nil
}

func Search(title string) (string, error) {
	ldb, err := leveldb.OpenFile("imdb.leveldb", &opt.Options{
		// Ideally we'd use zstd here, but it's not available.
		Compression: opt.SnappyCompression,
		// Database should already exist
		ErrorIfExist:   false,
		ErrorIfMissing: true,
		// We won't do any writes here.
		ReadOnly: true,
	})
	if err != nil {
		return "", err
	}
	entry, err := findTitle(ldb, title)
	if err != nil {
		return "", err
	}
	// Episodes aren't going to be sorted in any canonical order
	// by the database, so do it manually.
	slices.SortFunc(entry.Episodes, func(a, b *db.Title) int {
		if a.SeasonNumber == b.SeasonNumber {
			// nul
			return 0
		}
		if *a.SeasonNumber == *b.SeasonNumber {
			// Handle nil. It's impossible for them to be
			// equal if they're real pointers.
			if a.EpisodeNumber == b.EpisodeNumber {
				return 0
			}
			return *a.EpisodeNumber - *b.EpisodeNumber
		}
		return *a.SeasonNumber - *b.SeasonNumber
	})
	s, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	return string(s), nil
}
