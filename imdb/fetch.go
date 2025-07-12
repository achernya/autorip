package imdb

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cavaliergopher/grab/v3"
	"github.com/dustin/go-humanize"
)

const (
	datasetSource = "https://datasets.imdbws.com/"
	basics        = "title.basics.tsv.gz"
	episodes      = "title.episode.tsv.gz"
	ratings       = "title.ratings.tsv.gz"
)

var (
	desiredFiles = [...]string{
		// Basic information about the content, including its unique identifier and title.
		basics,
		// Association between a `tvSeries` and a `tvEpisode`.
		episodes,
		// Ratings, for better ranking of results
		ratings,
	}
)

// Fetch downloads all IMDb metadata that is needed for the index.
func Fetch(ctx context.Context, dir string) error {
	client := grab.NewClient()
	// For some reason, it looks like AWS Cloudfront (which is the
	// CDN for IMDb) does something weird if compression is
	// enabled between the HEAD and GET requests. So, just disable
	// it.
	client.HTTPClient.(*http.Client).Transport.(*http.Transport).DisableCompression = true

	requests := make([]*grab.Request, 0, len(desiredFiles))
	for _, file := range desiredFiles {
		r, err := grab.NewRequest(dir, datasetSource+file)
		if err != nil {
			return err
		}
		r.WithContext(ctx)
		requests = append(requests, r)
	}

	respch := client.DoBatch(4, requests...)
	for resp := range respch {
		if err := resp.Err(); err != nil {
			return err
		}
		fmt.Printf("Downloaded %s to %s (%s/s)\n", resp.Request.URL(), resp.Filename, humanize.Bytes(uint64(resp.BytesPerSecond())))
	}
	return nil
}
