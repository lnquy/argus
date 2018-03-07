package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	argus "github.com/lnquy/argus/lib"
	"github.com/lnquy/argus/lib/github"
	"github.com/lnquy/argus/lib/gitlab"
	"github.com/sirupsen/logrus"
)

type Report struct {
	Date    time.Time `json:"date"`
	Commits int       `json:"commits"`
}

var (
	contribs map[string]int
	log      *logrus.Entry

	fDebug = flag.Bool("d", false, "enabled debug log")
	fOut   = flag.String("o", "", "path to write output file")
)

func main() {
	flag.Parse()
	if *fDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	log = logrus.WithField("cmd", "argus")

	c := make([]argus.Crawler, 0)
	c = append(c, github.NewCrawler(&argus.SVC{
		User:   "lnquy",
		Emails: []string{"lnquy.it@gmail.com"},
		APIKey: "",
	}))
	c = append(c, gitlab.NewCrawler(&argus.SVC{
		User:   "lnquy",
		Emails: []string{"lnquy.it@gmail.com"},
		APIKey: "",
	}))

	wg := sync.WaitGroup{}
	repos := make([]argus.Repo, 0)
	reposChan := make(chan []argus.Repo, 10)
	for i := range c {
		wg.Add(1)
		go func(i int) {
			log.Infof("start fetching contributions for crawler #%d", i)
			defer wg.Done()
			r, err := c[i].GetContributions()
			if err != nil {
				log.Errorf("failed to fetch contribution for crawler #%d: %s", i, err)
				return
			}
			log.Infof("contributions for crawler #%d fetched", i)
			reposChan <- r
		}(i)
	}

	go func() {
		for rs := range reposChan {
			repos = append(repos, rs...)
		}
	}()
	wg.Wait()
	close(reposChan)
	time.Sleep(500 * time.Millisecond)

	b, _ := json.Marshal(repos)
	w := bytes.Buffer{}
	w.WriteString(fmt.Sprintf("Raw repos: \n%s", string(b)))
	w.WriteString(fmt.Sprintf("\n\nContributions by repo:\n"))

	for _, r := range repos {
		w.WriteString(fmt.Sprintf("%s (%s) - %d\n", r.Name, r.URL, len(r.Commits)))
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
	w.WriteString(fmt.Sprintf("\n\n\nContributions: \n%s", string(b)))

	fp := "argus.log"
	if *fOut != "" {
		fp = *fOut
	}
	if err := ioutil.WriteFile(fp, w.Bytes(), 0644); err != nil {
		log.Errorf("failed to write output file to %s: %s", fp, err)
	} else {
		log.Infof("output file written to %s", fp)
	}

	log.Info("exit")
}
