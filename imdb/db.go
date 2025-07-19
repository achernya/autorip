package imdb

import (
	"bufio"
	"cmp"
	"compress/gzip"
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	pb "github.com/achernya/autorip/proto"
)

const (
	batchSize   = 100000
	imdbLevelDb = "imdb.leveldb"
	imdbBleve   = "imdb.bleve"
)

// imdbTsv stores all of the components needed to read line-by-line
// from a tsv.gz. gzip.Reader does not Close the underlying io.Reader,
// and bufio.Scanner does not expose a Close method at all, so this
// struct bundles them together.
type imdbTsv struct {
	underlying *os.File
	gzipReader *gzip.Reader
	scanner    *bufio.Scanner
}

// newImdbTsv opens a tsv.gz file for reading.
func newImdbTsv(filename string) (*imdbTsv, error) {
	fgz, err := os.Open(filename)
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
	return &imdbTsv{
		underlying: fgz,
		gzipReader: f,
		scanner:    scanner,
	}, nil

}

// Close closes the tsv.gz file, including reporting gzip checksum errors.
func (i *imdbTsv) Close() error {
	if err := i.gzipReader.Close(); err != nil {
		return err
	}
	return i.underlying.Close()
}

func key(keys ...string) []byte {
	return []byte(strings.Join(keys, ","))
}

func decode(b []byte) (*pb.Title, error) {
	entry := &pb.Title{}
	if err := proto.Unmarshal(b, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

// lookup finds the given key in the given leveldb database and
// returns it as a decoded Title. No modification to the returned
// object will be performed (i.e., TConst is not filled in).
func lookup(ldb *leveldb.DB, title ...string) (*pb.Title, error) {
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

func episodesSort(a, b *pb.Title) int {
	if !a.HasSeasonNumber() || !b.HasSeasonNumber() {
		// Incomparable
		return 0
	}
	if a.GetSeasonNumber() == b.GetSeasonNumber() {
		if !a.HasEpisodeNumber() || !b.HasEpisodeNumber() {
			// Incomparable
			return 0
		}
		return cmp.Compare(a.GetEpisodeNumber(), b.GetEpisodeNumber())
	}
	return cmp.Compare(a.GetSeasonNumber(), b.GetSeasonNumber())
}

// Index is a LevelDB and Blevesearch index for the IMDB data.
type Index struct {
	dir   string
	ldb   *leveldb.DB
	index bleve.Index
}

func (i *Index) openLevelDb(read bool) error {
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
	ldb, err := leveldb.OpenFile(path.Join(i.dir, imdbLevelDb), options)
	if err != nil {
		return err
	}
	i.ldb = ldb
	return nil
}

// NewIndex prepares an index for population. If you want to query the
// index, use OpenIndex instead.
func NewIndex(dir string) (*Index, error) {
	idx := &Index{
		dir: dir,
	}
	if err := idx.openLevelDb(false); err != nil {
		return nil, err
	}
	mapping := bleve.NewIndexMapping()
	mapping.DefaultField = "Title"
	mapping.TypeField = "type"
	mapping.DefaultAnalyzer = "en"
	mapping.ScoringModel = "bm25"
	index, err := bleve.New(path.Join(idx.dir, imdbBleve), mapping)
	if err != nil {
		idx.ldb.Close()
		return nil, err
	}
	idx.index = index
	return idx, nil
}

// OpenIndex opens an index for queries. If you want to create a new
// index, use NewIndex instead.
func OpenIndex(dir string) (*Index, error) {
	idx := &Index{
		dir: dir,
	}
	if err := idx.openLevelDb(true); err != nil {
		return nil, err
	}
	index, err := bleve.OpenUsing(path.Join(dir, imdbBleve), map[string]interface{}{
		"read_only": true,
	})
	if err != nil {
		idx.ldb.Close()
		return nil, err
	}
	idx.index = index
	return idx, nil

}

func (i *Index) findTitle(title string) (*pb.Title, error) {
	it := i.ldb.NewIterator(&util.Range{key(title), nil}, nil)
	if !it.First() {
		return nil, fmt.Errorf("could not find key %+v", title)
	}
	var entry *pb.Title = nil
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
			entry.SetTConst(currKey[0])
			entry.SetEpisodes(make([]*pb.Title, 0))
		case 2:
			if entry == nil {
				return nil, fmt.Errorf("encountered subrecord before parent record for key %+v", title)
			}
			subentry, err := decode(it.Value())
			if err != nil {
				return nil, err
			}
			subentry.SetTConst(currKey[1])
			entry.SetEpisodes(append(entry.GetEpisodes(), subentry))
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
	for _, episode := range entry.GetEpisodes() {
		subentry, err := lookup(i.ldb, episode.GetTConst())
		if err != nil {
			return nil, err
		}
		proto.Merge(episode, subentry)
	}
	// Always return the episodes sorted, even though they're not stored that way.
	eps := entry.GetEpisodes()
	slices.SortFunc(eps, episodesSort)
	entry.SetEpisodes(eps)
	return entry, nil
}

func (i *Index) loadTitles() error {
	scanner, err := newImdbTsv(path.Join(i.dir, basics))
	if err != nil {
		return err
	}
	defer scanner.Close()

	tx, err := i.ldb.OpenTransaction()
	if err != nil {
		return err
	}
	count := 0
	batch := leveldb.Batch{}

	for scanner.scanner.Scan() {
		line := scanner.scanner.Text()
		record := strings.Split(line, "\t")
		title := pb.Title_builder{
			TitleType:     proto.String(record[1]),
			PrimaryTitle:  proto.String(record[2]),
			OriginalTitle: proto.String(record[3]),
			Genres:        strings.Split(record[8], ","),
		}.Build()
		if b, err := strconv.ParseBool(record[4]); err == nil {
			title.SetIsAdult(b)
		}
		if year, err := strconv.Atoi(record[5]); err == nil {
			title.SetStartYear(int32(year))
		}
		if year, err := strconv.Atoi(record[6]); err == nil {
			title.SetEndYear(int32(year))
		}
		if runtime, err := strconv.Atoi(record[7]); err == nil {
			title.SetRuntimeMinutes(int32(runtime))
		}
		b, err := proto.Marshal(title)
		if err != nil {
			return err
		}
		batch.Put(key(record[0]), b)
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

func (i *Index) loadEpisodes() error {
	scanner, err := newImdbTsv(path.Join(i.dir, episodes))
	if err != nil {
		return err
	}
	defer scanner.Close()

	tx, err := i.ldb.OpenTransaction()
	if err != nil {
		return err
	}
	count := 0
	batch := leveldb.Batch{}

	for scanner.scanner.Scan() {
		line := scanner.scanner.Text()
		record := strings.Split(line, "\t")
		tConst := record[0]
		parentTConst := record[1]
		// First, add the metadata to the episode
		update := &pb.Title{}
		seasonNumber, err := strconv.Atoi(record[2])
		if err == nil {
			update.SetSeasonNumber(int32(seasonNumber))
		}
		episodeNumber, err := strconv.Atoi(record[3])
		if err == nil {
			update.SetEpisodeNumber(int32(episodeNumber))
		}
		b, err := proto.Marshal(update)
		if err != nil {
			return err
		}
		batch.Put(key(parentTConst, tConst), b)
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

func (i *Index) makeSearch() error {
	scanner, err := newImdbTsv(path.Join(i.dir, ratings))
	if err != nil {
		return err
	}
	defer scanner.Close()

	count := 0
	batch := i.index.NewBatch()

	log.Println("Making search index")
	for scanner.scanner.Scan() {
		line := scanner.scanner.Text()
		record := strings.Split(line, "\t")
		if len(record) != 3 {
			return fmt.Errorf("got %+v, want 3 columns", line)
		}
		l, err := lookup(i.ldb, record[0])
		if err != nil {
			return err
		}
		entry := struct {
			Title         string
			AverageRating float32
			NumVotes      int
		}{
			Title: l.GetPrimaryTitle(),
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
			if err := i.index.Batch(batch); err != nil {
				return err
			}
			batch = i.index.NewBatch()
			count = 0
		}
	}
	log.Println("Done")
	return i.index.Batch(batch)
}

func (i *Index) makeLevelDb() error {
	// Load titles
	log.Println("Loading titles")
	if err := i.loadTitles(); err != nil {
		return err
	}

	// Load episodes
	log.Println("Loading episodes")
	if err := i.loadEpisodes(); err != nil {
		return err
	}

	log.Println("Compacting")
	if err := i.ldb.CompactRange(util.Range{nil, nil}); err != nil {
		return err
	}
	log.Println("Done")
	return nil
}

func (i *Index) Build() error {
	if err := i.makeLevelDb(); err != nil {
		return err
	}
	if err := i.makeSearch(); err != nil {
		return err
	}
	return nil
}

func (i *Index) Search(ctx context.Context, query string) (<-chan *pb.Result, error) {
	searchRequest := bleve.NewSearchRequest(bleve.NewQueryStringQuery(query))
	// For now, hard code the maximum results we're willing to
	// return to 100. In an ideal world, we'd paginate 10 at a
	// time and remove this limit, but realistically this is more
	// than anyone will ever need.
	searchRequest.Size = 100
	searchRequest.Fields = []string{"NumVotes", "AverageRating"}
	searchRequest.SortBy([]string{"-_score", "-NumVotes"})
	searchResult, err := i.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	ch := make(chan *pb.Result)
	go func() {
		for _, hit := range searchResult.Hits {
			result := &pb.Result{}
			entry, err := i.findTitle(hit.ID)
			if err != nil {
				log.Println(err)
				break
			}

			result.SetEntry(entry)
			result.SetScore(hit.Score)
			result.SetNumVotes(int32(hit.Fields["NumVotes"].(float64)))
			result.SetAverageRating(float32(hit.Fields["AverageRating"].(float64)))
			select {
			case <-ctx.Done():
				break
			case ch <- result:
			}
		}
		close(ch)

	}()
	return ch, nil
}

func (i *Index) SearchJSON(query string, maxResults int) (string, error) {
	results := &pb.Results{}
	results.SetResult(make([]*pb.Result, 0))

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := i.Search(ctx, query)
	if err != nil {
		return "", err
	}
	for len(results.GetResult()) < maxResults {
		result, ok := <-ch
		if !ok {
			break
		}
		results.SetResult(append(results.GetResult(), result))
	}
	cancel()

	s, err := protojson.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(s), nil
}

func (i *Index) Close() {
	if i == nil {
		return
	}
	if i.index != nil {
		i.index.Close()
	}
	if i.ldb != nil {
		i.ldb.Close()
	}
}
