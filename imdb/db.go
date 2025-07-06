package imdb

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"dario.cat/mergo"
	"github.com/achernya/autorip/db"
	"github.com/blevesearch/bleve/v2"
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

func episodesSort(a, b *db.Title) int {
	if a.SeasonNumber == nil {
		if b.SeasonNumber == nil {
			return 0
		}
		return -1
	}
	if b.SeasonNumber == nil {
		return 1
	}
	if *a.SeasonNumber == *b.SeasonNumber {
		if a.EpisodeNumber == nil {
			if b.EpisodeNumber == nil {
				return 0
			}
			return -1
		}
		if b.EpisodeNumber == nil {
			return 1
		}
		return *a.EpisodeNumber - *b.EpisodeNumber
	}
	return *a.SeasonNumber - *b.SeasonNumber
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
	// Always return the episodes sorted, even though they're not stored that way.
	slices.SortFunc(entry.Episodes, episodesSort)
	return entry, nil
}

func imdbScanner(dir string, filename string) (*bufio.Scanner, error) {
	fgz, err := os.Open(path.Join(dir, filename))
	if err != nil {
		return nil, err
	}
	f, err := gzip.NewReader(fgz)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(f)
	// Consume the header
	scanner.Scan()
	return scanner, nil
}

func loadTitles(ldb *leveldb.DB, dir string) error {
	scanner, err := imdbScanner(dir, basics)
	if err != nil {
		return err
	}

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
	scanner, err := imdbScanner(dir, episodes)
	if err != nil {
		return err
	}

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

func makeSearch(dir string) error {
	ldb, err := openLevelDb(true)
	if err != nil {
		return err
	}

	mapping := bleve.NewIndexMapping()
	mapping.DefaultField = "Title"
	index, err := bleve.New("imdb.bleve", mapping)
	if err != nil {
		return err
	}

	scanner, err := imdbScanner(dir, ratings)
	if err != nil {
		return err
	}

	count := 0
	batch := index.NewBatch()

	log.Println("Making search index")
	for scanner.Scan() {
		line := scanner.Text()
		record := strings.Split(line, "\t")
		if len(record) != 3 {
			return fmt.Errorf("got %+v, want 3 columns", line)
		}
		l, err := lookup(ldb, record[0])
		if err != nil {
			return err
		}
		entry := struct {
			Title         string
			AverageRating float32
			NumVotes      int
		}{
			Title: l.PrimaryTitle,
		}
		rating, err := strconv.ParseFloat(record[1], 32)
		if err != nil {
			return err
		}
		entry.AverageRating = float32(rating)
		votes, err := strconv.Atoi(record[2])
		if err != nil {
			return err
		}
		entry.NumVotes = votes
		batch.Index(record[0], entry)
		count++
		if count == batchSize {
			if err := index.Batch(batch); err != nil {
				return err
			}
			batch = index.NewBatch()
			count = 0
		}
	}
	log.Println("Done")
	return index.Batch(batch)
}

func openLevelDb(read bool) (*leveldb.DB, error) {
	options := &opt.Options{
		// Ideally we'd use zstd here, but it's not available.
		Compression: opt.SnappyCompression,
	}
	if read {
		// Database should already exist
		options.ErrorIfExist = false
		options.ErrorIfMissing = true
		// We won't do any writes here.
		options.ReadOnly = true
	} else {
		// We expect to create the database here.
		options.ErrorIfExist = true
		options.ErrorIfMissing = false
	}
	return leveldb.OpenFile("imdb.leveldb", options)
}

func openBleve(dir string) (bleve.Index, error) {
	return bleve.Open("imdb.bleve")
}

func makeImdb(dir string) error {
	ldb, err := openLevelDb(false)
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

func MakeIndex(dir string) error {
	if err := makeImdb(dir); err != nil {
		return err
	}
	if err := makeSearch(dir); err != nil {
		return err
	}
	return nil
}

func Search(title string) (string, error) {
	ldb, err := openLevelDb(true)
	if err != nil {
		return "", err
	}
	index, err := openBleve(".")
	if err != nil {
		return "", err
	}

	searchRequest := bleve.NewSearchRequest(bleve.NewQueryStringQuery(title))
	searchRequest.Fields = []string{"NumVotes", "AverageRating"}
	searchRequest.SortBy([]string{"-_score", "-NumVotes"})
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return "", err
	}
	if len(searchResult.Hits) == 0 {
		return "", fmt.Errorf("no results for %+v", title)
	}

	results := make([]map[string]any, 0)
	maxResults := min(len(searchResult.Hits), 10)

	for i := range maxResults {
		result := make(map[string]any)

		entry, err := findTitle(ldb, searchResult.Hits[i].ID)
		if err != nil {
			return "", err
		}

		result["Entry"] = entry
		result["Score"] = searchResult.Hits[i].Score
		maps.Copy(result, searchResult.Hits[i].Fields)
		results = append(results, result)
	}

	s, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(s), nil
}
