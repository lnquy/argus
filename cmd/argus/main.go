package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/labstack/gommon/log"
	argus "github.com/lnquy/argus/lib"
	//"github.com/lnquy/argus/lib/github"
	"github.com/lnquy/argus/lib/gitlab"
)

type Report struct {
	Date    time.Time `json:"date"`
	Commits int       `json:"commits"`
}

var (
	contribs map[string]int
)

func main() {
	//c := github.NewCrawler(&argus.SVC{
	//	User:   "lnquy",
	//	Emails:  []string{"lnquy.it@gmail.com"},
	//	APIKey: "",
	//})

	c := gitlab.NewCrawler(&argus.SVC{
		User:   "lnquy",
		Emails:  []string{"lnquy.it@gmail.com"},
		APIKey: "",
	})

	repos, err := c.GetContributions()
	if err != nil {
		log.Errorf("main: failed to fetch Github contributions: %s\n", err)
		return
	}
	b, _ := json.Marshal(repos)
	fmt.Printf("Raw repos: \n%s", string(b))

	contribs = make(map[string]int)
	for _, r := range repos {
		for _, c := range r.Commits {
			d := c.Date.Format("2006-01-02")
			if v, ok := contribs[d]; !ok {
				contribs[d] = 1
			} else {
				contribs[d] = v + 1
			}
		}
	}

	// Fill no commit days
	//now := time.Now().Unix()
	//today := now - (now % 86400) // To midnight ts
	//for i := 0; i < 375; i++ {
	//	ds := time.Unix(today, 0).Format("2006-01-02")
	//	if _, ok := contribs[ds]; !ok {
	//		contribs[ds] = 0
	//	}
	//	today -= 86400
	//}

	b, _ = json.Marshal(contribs)
	fmt.Printf("\n\n\nContributions: \n%s", string(b))
}
