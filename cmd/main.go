package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/labstack/gommon/log"
	argus "github.com/lnquy/argus/lib"
	"github.com/lnquy/argus/lib/github"
)

var (
	contribs map[int64]int
)

func main() {
	c := github.NewCrawler(&argus.SVC{
		User:   "lnquy",
		Email:  []string{"lnquy.it@gmail.com"},
		APIKey: "",
	})

	repos, err := c.GetContributions()
	if err != nil {
		log.Errorf("main: failed to fetch Github contributions: %s\n", err)
		return
	}
	b, _ := json.Marshal(repos)
	fmt.Printf("Raw repos: \n%s", string(b))

	contribs = make(map[int64]int)
	for _, r := range repos {
		for _, c := range r.Commits {
			ts := c.Date.Unix() - (c.Date.Unix() % 86400) // To midnight
			if v, ok := contribs[ts]; !ok {
				contribs[ts] = 1
			} else {
				contribs[ts] = v + 1
			}
		}
	}

	// Fill no commit days
	now := time.Now().Unix()
	today := now - (now % 86400) // To midnight ts
	for i := 0; i < 375; i++ {
		log.Printf("date: %s\n", today)
		if _, ok := contribs[today]; !ok {
			contribs[today] = 0
		}
		today -= 86400
	}

	b, _ = json.Marshal(contribs)
	fmt.Printf("\n\n\nContributions: \n%s", string(b))
}
