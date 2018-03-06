package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	argus "github.com/lnquy/argus/lib"
	"github.com/lnquy/argus/lib/github"
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
	c := make([]argus.Crawler, 0)
	c = append(c, github.NewCrawler(&argus.SVC{
		User:   "lnquy",
		Emails:  []string{"lnquy.it@gmail.com"},
		APIKey: "",
	}))
	c = append(c, gitlab.NewCrawler(&argus.SVC{
		User:   "lnquy",
		Emails:  []string{"lnquy.it@gmail.com"},
		APIKey: "",
	}))

	wg := sync.WaitGroup{}
	repos := make([]argus.Repo, 0)
	reposChan := make(chan []argus.Repo, 10)
	for i := range c {
		wg.Add(1)
		go func(i int) {
			log.Printf("main: fetching contribution #%d\n", i)
			defer wg.Done()
			r, err := c[i].GetContributions()
			if err != nil {
				log.Printf("main: failed to fetch contribution: %s\n", err)
				return
			}
			reposChan <-r
		}(i)
	}

	go func() {
		for rs := range reposChan {
			for _, r := range rs { // TODO
				if r.Name == "" && r.FullName == "" && r.URL == "" {
					continue
				}
				repos = append(repos, r)
			}
		}
	}()
	wg.Wait()
	close(reposChan)
	time.Sleep(500*time.Millisecond)

	b, _ := json.Marshal(repos)
	fmt.Printf("Raw repos: \n%s", string(b))

	fmt.Printf("\n\nContributions by repo:\n")
	for _, r := range repos {
		fmt.Printf("%s (%s) - %d\n", r.Name, r.URL, len(r.Commits))
	}

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

	b, _ = json.Marshal(contribs)
	fmt.Printf("\n\n\nContributions: \n%s", string(b))
}
