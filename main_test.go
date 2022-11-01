package main

import (
	"fmt"
	"github.com/wetor/AnimeGo/pkg/anisource/bangumi"
	"github.com/wetor/AnimeGo/pkg/cache"
	"testing"
)

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
