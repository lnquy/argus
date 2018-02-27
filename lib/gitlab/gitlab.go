package gitlab

import argus "github.com/lnquy/argus/lib"

type crawler struct {
	svc *argus.SVC
}

func NewCrawler(svc *argus.SVC) argus.Crawler {
	return &crawler{
		svc: svc,
	}
}

func (c *crawler) GetContributions() []*argus.Day {
	// TODO
	return nil
}

