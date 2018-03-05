package gitlab

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	argus "github.com/lnquy/argus/lib"
)

const (
	GITLAB     = "https://gitlab.com/api/v3"
	actPushed  = "pushed"
	actCreated = "created"
	actMerged  = "merged"
)

var (
	reposMap map[int]*argus.Repo
)

type crawler struct {
	svc *argus.SVC
}

func NewCrawler(svc *argus.SVC) argus.Crawler {
	return &crawler{
		svc: svc,
	}
}

func (c *crawler) GetContributions() ([]argus.Repo, error) {
	reposMap = make(map[int]*argus.Repo)
	events, err := c.listEvents()
	if err != nil {
		return nil, err
	}

	for _, e := range events {
		if r, ok := reposMap[e.ProjectID]; ok { // If project details has been crawled then just append commit
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
			tr, err := c.getRepo(e.ProjectID)
			if err != nil {
				tr.Name = fmt.Sprintf("%d", e.ProjectID)
			}
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
			reposMap[e.ProjectID] = tr
		}
	}

	repos := make([]argus.Repo, 0)
	for _, r := range reposMap {
		repos = append(repos, *r)
	}
	return repos, nil
}

func (c *crawler) listEvents() ([]Event, error) {
	events := make([]Event, 0)
	lastYear := time.Now().AddDate(-1, 0, 0)

	page := 0
	for {
		page++
		log.Printf("gitlab: crawling event page #%d\n", page)
		req := fmt.Sprintf("%s/events?per_page=100&page=%d&after=%s&private_token=%s",
			GITLAB, page, lastYear.Format("2006-01-02"), c.svc.APIKey)
		resp, err := http.Get(req)
		if err != nil {
			log.Printf("gitlab: failed to fetch event page %s: %s\n", req, err)
			return nil, err
		}
		tmpEvents := make([]Event, 0)
		if err = json.NewDecoder(resp.Body).Decode(&tmpEvents); err != nil {
			resp.Body.Close()
			log.Printf("github: failed to decode commits for %s: %s\n", req, err)
			return nil, err
		}
		resp.Body.Close()

		for _, e := range tmpEvents {
			if strings.HasPrefix(e.ActionName, actPushed) ||
				strings.HasPrefix(e.ActionName, actPushed) ||
				strings.HasPrefix(e.ActionName, actMerged) {
				events = append(events, e)
			}
		}
		if len(tmpEvents) == 0 || lastYear.Unix() >= tmpEvents[len(tmpEvents)-1].CreatedAt.Unix() {
			break
		}
	}

	log.Printf("gitlab: done events crawling\n")
	return events, nil
}

func (c *crawler) getRepo(pid int) (*argus.Repo, error) {
	log.Printf("gitlab: get repo detail: %d\n", pid)
	resp, err := http.Get(fmt.Sprintf("%s/projects/%d?private_token=%s", GITLAB, pid, c.svc.APIKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	localRepo := LocalRepo{}
	if err = json.NewDecoder(resp.Body).Decode(&localRepo); err != nil {
		return nil, err
	}
	log.Printf("gitlab: done get repo detail: %d\n", pid)
	return &argus.Repo{
		Name:      localRepo.Name,
		FullName:  localRepo.NameWithNamespace,
		URL:       localRepo.WebURL,
		Private:   !localRepo.Public,
		CreatedAt: localRepo.CreatedAt,
		PushedAt:  localRepo.LastActivityAt,
	}, nil
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
