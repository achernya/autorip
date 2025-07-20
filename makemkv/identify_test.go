package makemkv

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	pb "github.com/achernya/autorip/proto"
)

func TestAspectsOf(t *testing.T) {
	tests := map[string]struct {
		input    *TitleInfo
		expected int
	}{
		"empty": {
			input: &TitleInfo{
				Streams: []StreamInfo{},
			},
			expected: 0,
		},
		"video only": {
			input: &TitleInfo{
				Streams: []StreamInfo{
					{
						GenericInfo: GenericInfo{
							Type: "Video",
						},
					},
				},
			},
			expected: 1,
		},
		"audio only": {
			input: &TitleInfo{
				Streams: []StreamInfo{
					{
						GenericInfo: GenericInfo{
							Type: "Audio",
						},
					},
				},
			},
			expected: 1,
		},
		"subtitles only": {
			input: &TitleInfo{
				Streams: []StreamInfo{
					{
						GenericInfo: GenericInfo{
							Type: "Subtitles",
						},
					},
				},
			},
			expected: 1,
		},
		"chapters only": {
			input: &TitleInfo{
				GenericInfo: GenericInfo{
					ChapterCount: "1",
				},
			},
			expected: 1,
		},
		"all together": {
			input: &TitleInfo{
				GenericInfo: GenericInfo{
					ChapterCount: "1",
				},
				Streams: []StreamInfo{
					{
						GenericInfo: GenericInfo{
							Type: "Video",
						},
					},
					{
						GenericInfo: GenericInfo{
							Type: "Audio",
						},
					},
					{
						GenericInfo: GenericInfo{
							Type: "Subtitles",
						},
					},
				},
			},
			expected: 4,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			i := &Identifier{}
			aspects := i.AspectsOf(tt.input)
			if len(aspects) != tt.expected {
				t.Errorf("got %+v, which has %d elements, want %d", aspects, len(aspects), tt.expected)
			}
		})
	}
}

func TestFilterDiscInfo(t *testing.T) {
	tests := map[string]struct {
		input    *DiscInfo
		expected map[int]struct{}
	}{
		"empty": {
			input:    &DiscInfo{},
			expected: map[int]struct{}{},
		},
		"single title": {
			input: &DiscInfo{
				Titles: []TitleInfo{
					{},
				},
			},
			expected: map[int]struct{}{
				0: struct{}{},
			},
		},
		"multiple titles with equal score": {
			input: &DiscInfo{
				Titles: []TitleInfo{
					{},
					{},
				},
			},
			expected: map[int]struct{}{
				0: struct{}{},
				1: struct{}{},
			},
		},
		"multiple titles, first higher ": {
			input: &DiscInfo{
				Titles: []TitleInfo{
					{
						GenericInfo: GenericInfo{
							ChapterCount: "1",
						},
					},
					{},
				},
			},
			expected: map[int]struct{}{
				0: struct{}{},
			},
		},
		"multiple titles, second higher ": {
			input: &DiscInfo{
				Titles: []TitleInfo{
					{},
					{
						GenericInfo: GenericInfo{
							ChapterCount: "1",
						},
					},
				},
			},
			expected: map[int]struct{}{
				1: struct{}{},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			i := &Identifier{}
			titles := i.FilterDiscInfo(tt.input)
			for key := range titles {
				_, present := tt.expected[key]
				if present {
					continue
				}
				t.Errorf("got title at index %d, want none", key)
			}
			for key := range tt.expected {
				_, present := titles[key]
				if !present {
					t.Errorf("missing title at index %d", key)
				}
			}
		})
	}
}

func TestDiscLikelyContains(t *testing.T) {
	tests := map[string]struct {
		input    map[int]*TitleInfo
		expected []Score
	}{
		"empty": {
			input:    map[int]*TitleInfo{},
			expected: []Score{},
		},
		"movie": {
			input: map[int]*TitleInfo{
				0: &TitleInfo{
					GenericInfo: GenericInfo{
						Duration: "3:14:15",
					},
				},
			},
			expected: []Score{{Type: "movie", TitleIndex: 0}},
		},
		"short episode": {
			input: map[int]*TitleInfo{
				0: &TitleInfo{
					GenericInfo: GenericInfo{
						Duration: "0:22:00",
					},
				},
			},
			expected: []Score{{Type: "tvEpisode", TitleIndex: 0}},
		},
		"movie with extras": {
			input: map[int]*TitleInfo{
				0: &TitleInfo{
					GenericInfo: GenericInfo{
						Duration: "0:05:00",
					},
				},
				1: &TitleInfo{
					GenericInfo: GenericInfo{
						Duration: "0:06:00",
					},
				},
				2: &TitleInfo{
					GenericInfo: GenericInfo{
						Duration: "3:14:15",
					},
				},
			},
			expected: []Score{
				{Type: "movie", TitleIndex: 2},
				{Type: "tvEpisode", TitleIndex: 1},
				{Type: "tvEpisode", TitleIndex: 0},
			},
		},
		"movie with aspects": {
			input: map[int]*TitleInfo{
				0: &TitleInfo{
					GenericInfo: GenericInfo{
						Duration: "3:14:15",
					},
				},
				1: &TitleInfo{
					GenericInfo: GenericInfo{
						Duration: "3:14:15",
					},
				},
				2: &TitleInfo{
					GenericInfo: GenericInfo{
						Duration: "3:14:15",
					},
				},
			},
			expected: []Score{
				{Type: "movie", TitleIndex: 0},
				{Type: "movie", TitleIndex: 1},
				{Type: "movie", TitleIndex: 2},
			},
		},
		"movie with extended version": {
			input: map[int]*TitleInfo{
				0: &TitleInfo{
					GenericInfo: GenericInfo{
						Duration: "3:14:15",
					},
				},
				1: &TitleInfo{
					GenericInfo: GenericInfo{
						Duration: "3:24:15",
					},
				},
			},
			expected: []Score{{Type: "movie", TitleIndex: 1}, {Type: "movie", TitleIndex: 0}},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			i := &Identifier{}
			scores, err := i.DiscLikelyContains(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			length := min(len(tt.expected), len(scores))
			if len(tt.expected) != len(scores) {
				t.Errorf("got %d scores, want %d", len(scores), len(tt.expected))
			}
			for i := range scores {
				if i >= length {
					break
				}
				if scores[i].Type != tt.expected[i].Type {
					t.Errorf("at index %d got %s, want %s", i, scores[i].Type, tt.expected[i].Type)
				}
				if scores[i].TitleIndex != tt.expected[i].TitleIndex {
					t.Errorf("at index %d got %d, want %d", i, scores[i].TitleIndex, tt.expected[i].TitleIndex)
				}
			}
		})
	}
}

type fakeIndex struct {
	results []*pb.Result
}

func (f *fakeIndex) Build() error {
	return nil
}

func (f *fakeIndex) Search(ctx context.Context, query string) (<-chan *pb.Result, error) {
	// Make the channel buffered so we don't need to spawn a
	// goroutine just to stuff in the result.
	ch := make(chan *pb.Result, len(f.results))
	for _, result := range f.results {
		ch <- result
	}
	close(ch)
	return ch, nil
}

func (f *fakeIndex) SearchJSON(query string, maxResults int) (string, error) {
	return "", nil
}

func (f *fakeIndex) Close() {
}

func TestXrefImdb(t *testing.T) {
	tests := map[string]struct {
		disc     *DiscInfo
		scores   []*Score
		index    *fakeIndex
		expected int
	}{
		"empty": {
			disc:     &DiscInfo{},
			scores:   []*Score{},
			index:    &fakeIndex{},
			expected: -1,
		},
		"one movie": {
			disc: &DiscInfo{},
			scores: []*Score{
				{
					Duration: 100 * time.Minute,
					Type:     "movie",
				},
			},
			index: &fakeIndex{
				results: []*pb.Result{
					pb.Result_builder{
						Entry: pb.Title_builder{
							TitleType:      proto.String("movie"),
							RuntimeMinutes: proto.Int32(100),
						}.Build(),
					}.Build(),
				},
			},
			expected: 0,
		},
		"unrelated results first": {
			disc: &DiscInfo{},
			scores: []*Score{
				{
					Duration: 100 * time.Minute,
					Type:     "movie",
				},
			},
			index: &fakeIndex{
				results: []*pb.Result{
					pb.Result_builder{
						Entry: pb.Title_builder{
							TitleType:      proto.String("tvEpisode"),
							RuntimeMinutes: proto.Int32(22),
						}.Build(),
					}.Build(),
					pb.Result_builder{
						Entry: pb.Title_builder{
							TitleType: proto.String("tvSeries"),
						}.Build(),
					}.Build(),
					pb.Result_builder{
						Entry: pb.Title_builder{
							TitleType:      proto.String("movie"),
							RuntimeMinutes: proto.Int32(100),
						}.Build(),
					}.Build(),
				},
			},
			expected: 2,
		},
		"wrong length first": {
			disc: &DiscInfo{},
			scores: []*Score{
				{
					Duration: 100 * time.Minute,
					Type:     "movie",
				},
			},
			index: &fakeIndex{
				results: []*pb.Result{
					pb.Result_builder{
						Entry: pb.Title_builder{
							TitleType:      proto.String("movie"),
							RuntimeMinutes: proto.Int32(80),
						}.Build(),
					}.Build(),
					pb.Result_builder{
						Entry: pb.Title_builder{
							TitleType:      proto.String("movie"),
							RuntimeMinutes: proto.Int32(100),
						}.Build(),
					}.Build(),
				},
			},
			expected: 1,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var want *pb.Title = nil
			if tt.expected != -1 {
				want = tt.index.results[tt.expected].GetEntry()
			}
			i := NewIdentifier(tt.index)
			got, err := i.XrefImdb(tt.disc, tt.scores)
			if err != nil {
				t.Fatal(err)
			}
			wantBytes, err := prototext.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			gotBytes, err := prototext.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(got, want) {
				t.Errorf("got %s, want %s", string(gotBytes), string(wantBytes))
			}
		})
	}
}
