package main

import (
	"fmt"
	"github.com/wetor/AnimeGo/pkg/anisource/bangumi"
	"github.com/wetor/AnimeGo/pkg/cache"
	"testing"
)

func TestDownload(t *testing.T) {
	url := "https://github.com/wetor/AnimeGo/releases/download/v0.3.0/AnimeGo_windows_amd64.zip"
	file := "./test.zip"
	download(url, -1, file)
}

func TestClean(t *testing.T) {
	result := CleanSubject("subject.jsonlines")
	fmt.Println(result)
	result = CleanEpisode("episode.jsonlines")
	fmt.Println(result)
	UpdateSubject()

	fmt.Println(SubjectMap[10380], EpisodeMap[10380])
	//result = SaveSubjectBolt(SubjectDB)
	//fmt.Println(result)
}

func TestGet(t *testing.T) {
	db := cache.NewBolt()
	db.Open("bolt_sub.db")

	sub := &bangumi.Entity{}
	err := db.Get(SubjectBucket, int64(10380), sub)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(sub)
}
