package db

type Title struct {
	TConst         string
	TitleType      string
	PrimaryTitle   string
	OriginalTitle  string
	IsAdult        bool
	StartYear      int
	EndYear        *int `json:",omitempty"`
	RuntimeMinutes int
	Genres         []string
	// Only populated for tvSeries
	Episodes []*Title `json:",omitempty"`
	// Only populated for tvEpisode
	SeasonNumber  *int `json:",omitempty"`
	EpisodeNumber *int `json:",omitempty"`
}
