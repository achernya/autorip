package makemkv

import (
	"cmp"
	"context"
	"log"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/achernya/autorip/imdb"

	pb "github.com/achernya/autorip/proto"
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

type Identifier struct {
	index imdb.GenericIndex
}

func NewIdentifier(index imdb.GenericIndex) *Identifier {
	return &Identifier{
		index: index,
	}
}

type Aspect struct {
	Score       int
	Description string
}

func (i *Identifier) AspectsOf(ti *TitleInfo) []Aspect {
	result := make([]Aspect, 0)
	hasVideo := false
	hasAudio := false
	hasSubtitles := false
	for _, si := range ti.Streams {
		hasVideo = hasVideo || si.Type == "Video"
		hasAudio = hasAudio || si.Type == "Audio"
		hasSubtitles = hasSubtitles || si.Type == "Subtitles"
	}
	if hasVideo {
		result = append(result, Aspect{
			Score:       0x80,
			Description: "has at least 1 video stream",
		})
	}
	if hasAudio {
		result = append(result, Aspect{
			Score:       0x40,
			Description: "has at least 1 audio stream",
		})

	}
	if hasSubtitles {
		result = append(result, Aspect{
			Score:       0x20,
			Description: "has at least 1 audio stream",
		})
	}
	if len(ti.ChapterCount) > 0 {
		result = append(result, Aspect{
			Score:       0x10,
			Description: "has chapters",
		})
	}
	return result
}

func scoreAspects(aspects []Aspect) int {
	sum := 0
	for _, aspect := range aspects {
		sum += aspect.Score
	}
	return sum
}

// FilterDiscInfo will return a map of {index, TitleInfo} of all
// titles that are likely to contain "main features". This is
// calculated by looking at the aspects of the different titles, and
// giving them each a score. Aspects analyzed include having a video
// track, audio track, and subtitles track, as well as having chapter
// markers. A title with the highest score is most likely the main feature.
//
// It is possible (and in the case of a tvSeries, likely) that there
// will be multiple titles that have these properties.
func (i *Identifier) FilterDiscInfo(di *DiscInfo) map[int]*TitleInfo {
	result := make(map[int]*TitleInfo)
	for index, ti := range di.Titles {
		result[index] = &ti
	}
	aspects := make(map[int][]Aspect)
	maxScore := 0
	for index, ti := range result {
		aspects[index] = i.AspectsOf(ti)
		score := scoreAspects(aspects[index])
		if score > maxScore {
			maxScore = score
		}
	}
	for index := range result {
		score := scoreAspects(aspects[index])
		if score == maxScore {
			continue
		}
		log.Printf("Removing index %d, score %d < %d\n", index, score, maxScore)
		delete(result, index)
	}
	return result

}

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
		// This is a static constant...if we can't parse this,
		// we have bigger issues and need to abort.
		panic(err)
	}
	return t.Sub(sub), nil
}

// DiscLikelyContains returns a sorted-descending list containing a
// score, type, and index for the titles on the disc. Note that this
// function will return "tvEpisode", not "tvSeries" as it's
// identifying the content on the disc.
//
// The input to this function should be the filtered map produced by
// FilterDiscInfo.
func (i *Identifier) DiscLikelyContains(titles map[int]*TitleInfo) ([]*Score, error) {
	scores := make([]*Score, 0)
	for index, title := range titles {
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
			if a.Likelihood == b.Likelihood {
				// If two things are equally likely,
				// then we should defer to the
				// index. However, since we're going
				// to reverse the scores after
				// sorting, flip the order here so the
				// lower index is first.
				return cmp.Compare(b.TitleIndex, a.TitleIndex)
			}
			return cmp.Compare(a.Likelihood, b.Likelihood)
		}
		return cmp.Compare(a.Duration, b.Duration)
	})
	slices.Reverse(scores)
	for _, score := range scores {
		log.Printf("title %d likely %s (score=%f) [%s]\n", score.TitleIndex, score.Type, score.Likelihood, score.Duration)
	}
	return scores, nil
}

func (i *Identifier) XrefImdb(di *DiscInfo, scores []*Score) (*pb.Title, error) {
	// Some discs have a name that is made entirely of spaces. So
	// remove leading/trailing spaces.
	name := strings.TrimSpace(di.Name)
	// and if it's empty, replace it with the VolumeName.
	if len(name) == 0 {
		name = di.VolumeName
	}
	// volume names have '_' instead of ' ', but we need the
	// search terms to be seperated by spaces to work well.
	query := strings.ReplaceAll(name, "_", " ")
	// Also escape `:` since that will be a field selector
	query = strings.ReplaceAll(query, ":", "\\:")
	// Also escape `-` since that is a negation character.
	query = strings.ReplaceAll(query, "-", "\\-")
	log.Printf("Searching %+q\n", query)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := i.index.Search(ctx, query)
	if err != nil {
		cancel()
		return nil, err
	}
	defer cancel()
	for info := range ch {
		// For now, we'll use a very simple algorithm: assume
		// that the classifier for movie vs tvEpisode was
		// correct, then find the first entry that has a
		// "similar enough" runtime. That should be enough to
		// distinguish between remakes.
		entry := info.GetEntry()
		if entry.GetTitleType() == "tvEpisode" {
			// The search engine returns individual
			// episodes, but we're interested in the
			// container series at this point.
			continue
		}
		if entry.GetTitleType() != scores[0].Type {
			log.Printf("Skipping %s (got %s, want %s)\n", entry.GetTConst(), entry.GetTitleType(), scores[0].Type)
			continue
		}
		durations := []time.Duration{time.Minute * time.Duration(entry.GetRuntimeMinutes()), scores[0].Duration}
		slices.Sort(durations)
		ratio := float64(durations[0]) / float64(durations[1])
		if ratio > 0.975 {
			log.Printf("Found [%s] %s\n", entry.GetTConst(), entry.GetPrimaryTitle())
			return entry, nil
		}
		log.Printf("Skipping %s, bad ratio %f\n", entry.GetPrimaryTitle(), ratio)
	}
	return nil, nil
}

type Plan struct {
	Identity  *pb.Title
	DiscInfo  *DiscInfo
	RipTitles []*Score
}

func (i *Identifier) MakePlan(discInfo *DiscInfo) (*Plan, error) {
	titles := i.FilterDiscInfo(discInfo)
	likely, err := i.DiscLikelyContains(titles)
	if err != nil {
		return nil, err
	}
	identity, err := i.XrefImdb(discInfo, likely)
	if err != nil {
		return nil, err
	}
	result := &Plan{
		Identity:  identity,
		DiscInfo:  discInfo,
		RipTitles: likely,
	}
	if identity.GetTitleType() == "movie" {
		// For a movie, only the first title will be
		// ripped.

		// TODO(achernya): deal with the Inception edge case
		// here and in XrefImdb.
		result.RipTitles = result.RipTitles[:1]
	}
	return result, nil
}
