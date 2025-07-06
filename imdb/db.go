package imdb

import (
	"compress/gzip"
	"encoding/json"
	"bufio"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/achernya/autorip/db"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func loadTitles(d *gorm.DB, dir string) error {
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
	
	tx := d.Begin()
	for scanner.Scan() {
		line := scanner.Text()
		record := strings.Split(line, "\t")
		title := db.Title{
			TConst:        record[0],
			TitleType:     record[1],
			PrimaryTitle:  record[2],
			OriginalTitle: record[3],
			Genres:        datatypes.NewJSONSlice(strings.Split(record[8], ",")),
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
		result := tx.Create(&title)
		if result.Error != nil {
			return result.Error
		}
	}
	return tx.Commit().Error
}

func loadEpisodes(d *gorm.DB, dir string) error {
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

	tx := d.Begin()
	for scanner.Scan() {
		line := scanner.Text()
		record := strings.Split(line, "\t")
		tConst := record[0]
		parentTConst := record[1]
		// First, add the metadata to the episode
		episode := db.Title{}
		result := tx.Select("ID").Where("t_const = ?", tConst).First(&episode)
		if result.Error != nil {
			return result.Error
		}
		update := db.Title{}
		seasonNumber, err := strconv.Atoi(record[2])
		if err == nil {
			update.SeasonNumber = &seasonNumber
		}
		episodeNumber, err := strconv.Atoi(record[3])
		if err == nil {
			update.EpisodeNumber = &episodeNumber
		}
		tx.Model(&episode).Updates(update)
		// Now, update the parent series with the association
		series := db.Title{}
		result = tx.Select("ID").Where("t_const = ?", parentTConst).First(&series)
		if result.Error != nil {
			return result.Error
		}
		if err := tx.Model(&series).Association("Episodes").Append(&episode); err != nil {
			return err
		}
	}
	return tx.Commit().Error
}

func MakeIndex(dir string) error {
	d, err := db.OpenImdb("imdb.sqlite")
	if err != nil {
		return err
	}
	// Drop the existing tables
	// log.Println("Dropping old tables")
	// d.Exec("DELETE FROM titles")
	// d.Exec("DELETE FROM episodes")

	// Load titles
	// log.Println("Loading titles")
	// if err := loadTitles(d, dir); err != nil {
	// 	return err
	// }

	// Load episodes
	log.Println("Loading episodes")
	if err := loadEpisodes(d, dir); err != nil {
		return err
	}
	log.Println("Done")
	return nil
}

func Search(title string) (string, error) {
	d, err := db.OpenImdb("imdb.sqlite")
	if err != nil {
		return "", err
	}
	entry := db.Title{}
	result := d.Preload("Episodes").Where("primary_title = ?", title).First(&entry)
	if result.Error != nil {
		return "", result.Error
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
