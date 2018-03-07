package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	argus "github.com/lnquy/argus/lib"
	"github.com/sirupsen/logrus"
)

const (
	GITLAB     = "https://gitlab.com/api/v3"
	actPushed  = "pushed"
	actCreated = "created"
	actMerged  = "merged"
)

type crawler struct {
	svc      *argus.SVC
	reposMap map[int]*argus.Repo
	reposMux sync.RWMutex
}

func NewCrawler(svc *argus.SVC) argus.Crawler {
	c := &crawler{
		svc: svc,
		reposMap: make(map[int]*argus.Repo),
		reposMux: sync.RWMutex{},
	}
	c.getLogger().Info("crawler created")
	return c
}

func (c *crawler) GetContributions() ([]argus.Repo, error) {
	log := c.getLogger()
	log.Info("start fetching contributions")
	events, err := c.listEvents()
	if err != nil {
		return nil, err
	}

	prjChan := make(chan int, 1000)
	wg := sync.WaitGroup{}
	c.getRepo(prjChan, &wg) // Run workers to crawl project details in background

	for _, e := range events {
		c.reposMux.Lock()
		if r, ok := c.reposMap[e.ProjectID]; ok { // If project details has been crawled then just append commit
			r.Commits = append(r.Commits, argus.Commit{
				Sha:  e.PushData.CommitTo,
				Date: e.CreatedAt,
				Author: argus.Author{
					Login: e.Author.Username,
					ID:    e.Author.ID,
					Name:  e.Author.Name,
					Email: "",
				},
			})
		} else { // Otherwise update repo details and its commits
			pid := e.ProjectID
			prjChan <- pid
			tr := argus.Repo{}
			tr.Commits = append(tr.Commits, argus.Commit{
				Sha:  e.PushData.CommitTo,
				Date: e.CreatedAt,
				Author: argus.Author{
					Login: e.Author.Username,
					ID:    e.Author.ID,
					Name:  e.Author.Name,
					Email: "",
				},
			})
			c.reposMap[pid] = &tr
		}
		c.reposMux.Unlock()
	}
	close(prjChan)
	wg.Wait()

	repos := make([]argus.Repo, 0)
	c.reposMux.RLock()

	//b, _ := json.MarshalIndent(c.reposMap, "", "  ")
	//fmt.Printf("MAP: \n\n%s\n\n", string(b))

	for _, r := range c.reposMap {
		repos = append(repos, *r)
	}
	c.reposMux.RUnlock()
	log.Info("done fetching contributions")
	return repos, nil
}

// Let 10 workers crawls event pages in parallel.
// Check at last page (page % 10 == 0), if there's no events then stop all workers and return result.
// Otherwise, continue crawling from (page+1) to (page+10)
func (c *crawler) listEvents() ([]Event, error) {
	log := c.getLogger()
	log.Debug("fetch all contributed events")
	events := make([]Event, 0)
	lastYear := time.Now().AddDate(-1, 0, 0)
	pageChan := make(chan int, 20)
	eventChan := make(chan Event, 1000)
	for i := 1; i <= 10; i++ {
		pageChan <- i
	}

	wg := sync.WaitGroup{}
	for r := 0; r < 10; r++ { // Workers
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			for p := range pageChan {
				log.Debugf("fetch event page #%d", p)
				req := fmt.Sprintf("%s/events?per_page=100&page=%d&after=%s&private_token=%s",
					GITLAB, p, lastYear.Format("2006-01-02"), c.svc.APIKey)
				resp, err := http.Get(req)
				if err != nil {
					log.Errorf("failed to fetch event page #%d: %s", p, err)
					continue
				}
				tmpEvents := make([]Event, 0)
				if err = json.NewDecoder(resp.Body).Decode(&tmpEvents); err != nil {
					resp.Body.Close()
					log.Debugf("failed to decode events for page #%d: %s", p, err)
					continue
				}
				resp.Body.Close()

				for _, e := range tmpEvents { // Report crawled events
					if strings.HasPrefix(e.ActionName, actPushed) ||
						strings.HasPrefix(e.ActionName, actCreated) ||
						strings.HasPrefix(e.ActionName, actMerged) {
						eventChan <- e
					}
				}
				log.Debugf("event page fetched for #%d", p)

				// Check at every 10th page.
				// If no events available then close the page channel and end worker.
				// Otherwise, distribute new jobs (page+1 -> page+10) to the page channel.
				if p%10 == 0 {
					if len(tmpEvents) == 0 || lastYear.Unix() >= tmpEvents[len(tmpEvents)-1].CreatedAt.Unix() {
						if pageChan != nil {
							close(pageChan)
						}
					} else {
						for i := p + 1; i <= p+10; i++ {
							pageChan <- i
						}
					}
				}
			}
			wg.Done()
		}(&wg)
	}

	// Put all crawled events to the array
	go func() {
		for e := range eventChan {
			events = append(events, e)
		}
	}()
	wg.Wait() // Wait for all workers returns
	close(eventChan)
	// Sleep a little bit so all events in eventChan can be consumed before returning
	time.Sleep(500 * time.Millisecond)

	log.Debug("all contributed events fetched")
	return events, nil
}

func (c *crawler) getRepo(pidChan chan int, wg *sync.WaitGroup) {
	log := c.getLogger()
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func() {
			for pid := range pidChan {
				log.Debugf("fetch repo detail: %d", pid)
				resp, err := http.Get(fmt.Sprintf("%s/projects/%d?private_token=%s", GITLAB, pid, c.svc.APIKey))
				if err != nil {
					continue
				}

				lr := LocalRepo{}
				if err = json.NewDecoder(resp.Body).Decode(&lr); err != nil {
					resp.Body.Close()
					continue
				}
				resp.Body.Close()

				c.reposMux.Lock()
				r := c.reposMap[pid]
				c.reposMap[pid] = &argus.Repo{
					Name:      lr.Name,
					FullName:  lr.NameWithNamespace,
					URL:       lr.WebURL,
					Private:   !lr.Public,
					CreatedAt: lr.CreatedAt,
					PushedAt:  lr.LastActivityAt,
					Commits:   r.Commits,
				}
				c.reposMux.Unlock()
				log.Debugf("repo detail fetched: %d - %s (%s)", pid, lr.Name, lr.WebURL)
			}
			wg.Done()
		}()
	}
}

func (c *crawler) getLogger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"type": "gitlab",
		"user": c.svc.User,
	})
}

type Event struct {
	ProjectID   int         `json:"project_id"`
	ActionName  string      `json:"action_name"`
	TargetID    interface{} `json:"target_id"`
	TargetIid   interface{} `json:"target_iid"`
	TargetType  interface{} `json:"target_type"`
	AuthorID    int         `json:"author_id"`
	TargetTitle interface{} `json:"target_title"`
	CreatedAt   time.Time   `json:"created_at"`
	Author struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		State     string `json:"state"`
		AvatarURL string `json:"avatar_url"`
		WebURL    string `json:"web_url"`
	} `json:"author"`
	PushData struct {
		CommitCount int         `json:"commit_count"`
		Action      string      `json:"action"`
		RefType     string      `json:"ref_type"`
		CommitFrom  interface{} `json:"commit_from"`
		CommitTo    string      `json:"commit_to"`
		Ref         string      `json:"ref"`
		CommitTitle string      `json:"commit_title"`
	} `json:"push_data"`
	AuthorUsername string `json:"author_username"`
}

type LocalRepo struct {
	ID                             int           `json:"id"`
	Description                    string        `json:"description"`
	DefaultBranch                  string        `json:"default_branch"`
	TagList                        []interface{} `json:"tag_list"`
	Public                         bool          `json:"public"`
	Archived                       bool          `json:"archived"`
	VisibilityLevel                int           `json:"visibility_level"`
	SSHURLToRepo                   string        `json:"ssh_url_to_repo"`
	HTTPURLToRepo                  string        `json:"http_url_to_repo"`
	WebURL                         string        `json:"web_url"`
	Name                           string        `json:"name"`
	NameWithNamespace              string        `json:"name_with_namespace"`
	Path                           string        `json:"path"`
	PathWithNamespace              string        `json:"path_with_namespace"`
	ResolveOutdatedDiffDiscussions bool          `json:"resolve_outdated_diff_discussions"`
	ContainerRegistryEnabled       bool          `json:"container_registry_enabled"`
	IssuesEnabled                  bool          `json:"issues_enabled"`
	MergeRequestsEnabled           bool          `json:"merge_requests_enabled"`
	WikiEnabled                    bool          `json:"wiki_enabled"`
	BuildsEnabled                  bool          `json:"builds_enabled"`
	SnippetsEnabled                bool          `json:"snippets_enabled"`
	CreatedAt                      time.Time     `json:"created_at"`
	LastActivityAt                 time.Time     `json:"last_activity_at"`
	SharedRunnersEnabled           bool          `json:"shared_runners_enabled"`
	LfsEnabled                     bool          `json:"lfs_enabled"`
	CreatorID                      int           `json:"creator_id"`
	Namespace struct {
		ID       int         `json:"id"`
		Name     string      `json:"name"`
		Path     string      `json:"path"`
		Kind     string      `json:"kind"`
		FullPath string      `json:"full_path"`
		ParentID interface{} `json:"parent_id"`
	} `json:"namespace"`
	AvatarURL                                 interface{}   `json:"avatar_url"`
	StarCount                                 int           `json:"star_count"`
	ForksCount                                int           `json:"forks_count"`
	OpenIssuesCount                           int           `json:"open_issues_count"`
	RunnersToken                              string        `json:"runners_token"`
	PublicBuilds                              bool          `json:"public_builds"`
	SharedWithGroups                          []interface{} `json:"shared_with_groups"`
	OnlyAllowMergeIfBuildSucceeds             bool          `json:"only_allow_merge_if_build_succeeds"`
	RequestAccessEnabled                      bool          `json:"request_access_enabled"`
	OnlyAllowMergeIfAllDiscussionsAreResolved bool          `json:"only_allow_merge_if_all_discussions_are_resolved"`
	Permissions struct {
		ProjectAccess interface{} `json:"project_access"`
		GroupAccess struct {
			AccessLevel       int `json:"access_level"`
			NotificationLevel int `json:"notification_level"`
		} `json:"group_access"`
	} `json:"permissions"`
}
