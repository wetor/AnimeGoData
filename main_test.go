package main

import (
	"fmt"
	"github.com/wetor/AnimeGo/pkg/cache"
	"testing"
)

func TestGet(t *testing.T) {
	db := cache.NewBolt()
	db.Open("bolt_sub.db")

	sub := &Entity{}
	err := db.Get(SubjectBucket, 10380, sub)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(sub)
}
