package makemkv

import (
	"cmp"
	"log"
	"math"
	"slices"
	"time"
)

type distribution struct {
	mean   float64
	stddev float64
}

var (
	// These distributions were calcuated with some throwaway code
	// on 2025-07-12 using the latest available IMDb data at the time.
	dists = map[string]distribution{
		"movie": {
			mean:   88.96,
			stddev: 27.35,
		},
		"tvEpisode": {
			mean:   39.04,
			stddev: 29.34,
		},
	}
)

type Score struct {
	TitleIndex int
	Duration   time.Duration
	Type       string
	Likelihood float64
}

func gaussianPdf(sample, mean, stddev float64) float64 {
	z := (sample - mean) / stddev
	return math.Exp(-z*z/2) / (stddev * math.Sqrt(2*math.Pi))
}

func parseHhMmSs(in string) (time.Duration, error) {
	t, err := time.Parse(time.TimeOnly, in)
	if err != nil {
		return 0, err
	}
	sub, err := time.Parse(time.TimeOnly, "00:00:00")
	if err != nil {
		panic(err)
	}
	return t.Sub(sub), nil
}

// DiscLikelyContains returns a sorted-descending list containing a
// score, type, and index for the titles on the disc. Note that this
// function will return "tvEpisode", not "tvSeries" as it's
// identifying the content on the disc.
func DiscLikelyContains(discInfo *DiscInfo) ([]*Score, error) {
	scores := make([]*Score, 0)
	for index, title := range discInfo.Titles {
		dur, err := parseHhMmSs(title.Duration)
		if err != nil {
			return nil, err
		}
		type nameAndValue struct {
			name  string
			value float64
		}
		result := make([]*nameAndValue, 0)
		for name, dist := range dists {
			result = append(result, &nameAndValue{
				name:  name,
				value: gaussianPdf(dur.Minutes(), dist.mean, dist.stddev),
			})
		}
		if len(result) == 0 {
			// No distributions?
			continue
		}
		slices.SortFunc(result, func(a, b *nameAndValue) int {
			return cmp.Compare(a.value, b.value)
		})
		// result[0] now contains the smallest PDF, and the
		// last element the maximum. The likelihood is the
		// ratio between the two.
		last := len(result) - 1
		scores = append(scores, &Score{
			TitleIndex: index,
			Duration:   dur,
			Type:       result[last].name,
			Likelihood: result[last].value / result[0].value,
		})
	}
	slices.SortFunc(scores, func(a, b *Score) int {
		if a.Duration == b.Duration {
			return cmp.Compare(a.Likelihood, b.Likelihood)
		}
		return cmp.Compare(a.Duration, b.Duration)
	})
	slices.Reverse(scores)
	if len(scores) > 0 {
		log.Printf("title %d likely %s (score=%f) [%s]\n", scores[0].TitleIndex, scores[0].Type, scores[0].Likelihood, scores[0].Duration)
	}
	return scores, nil
}
