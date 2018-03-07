package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	argus "github.com/lnquy/argus/lib"
	"github.com/sirupsen/logrus"
)

const githubAPI = "https://api.github.com"

type crawler struct {
	svc *argus.SVC
}

func NewCrawler(svc *argus.SVC) argus.Crawler {
	c := &crawler{
		svc: svc,
	}
	c.getLogger().Info("crawler created")
	return c
}

func (c *crawler) GetContributions() ([]argus.Repo, error) {
	log := c.getLogger()
	log.Info("start fetching contributions")
	repos, err := c.listRepos()
	if err != nil {
		return nil, err
	}
	log.Info("done fetching contributions")
	return repos, nil
}

func (c *crawler) listRepos() ([]argus.Repo, error) {
	log := c.getLogger()
	log.Debug("fetch all repos")
	repos := make([]argus.Repo, 0)

	resp, err := http.Get(fmt.Sprintf("%s/user/repos?sort=pushed&per_page=100&access_token=%s", githubAPI, c.svc.APIKey))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err = json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	for i := 0; i < len(repos); i++ {
		wg.Add(1)
		go c.listCommitsByRepo(&wg, &repos[i])
	}
	wg.Wait()
	log.Debug("all repos fetched")
	return repos, nil
}

func (c *crawler) listCommitsByRepo(wg *sync.WaitGroup, r *argus.Repo) {
	defer wg.Done()
	log := c.getLogger()
	log.Debugf("fetch commits for repo %s", r.URL)
	lastYear := time.Now().AddDate(-1, 0, 0)

	page := 0
	for {
		page++
		req := fmt.Sprintf("%s/repos/%s/commits?since=%s&author=%s&per_page=100&access_token=%s&page=%d",
			githubAPI, r.FullName, lastYear.Format("2006-01-02T03:04:05Z"), c.svc.User, c.svc.APIKey, page)
		resp, err := http.Get(req)
		if err != nil {
			log.Errorf("failed to fetch commits for %s: %s", req, err)
			return
		}
		commits := make([]commit, 0)
		if err = json.NewDecoder(resp.Body).Decode(&commits); err != nil {
			resp.Body.Close()
			log.Errorf("failed to decode commits for %s: %s", req, err)
			return
		}
		resp.Body.Close()

		r.Commits = append(r.Commits, toExportCommit(commits)...)
		// If last commits is older than 1 year ago then stop crawling commits on this repo
		if len(commits) == 0 || lastYear.Unix() >= commits[len(commits)-1].Commit.Author.Date.Unix() {
			break
		}
	}

	log.Debugf("commits fetched for %s", r.URL)
}

func toExportCommit(cs []commit) []argus.Commit {
	ret := make([]argus.Commit, 0)
	for _, c := range cs {
		ret = append(ret, argus.Commit{
			Sha:  c.Sha,
			Date: c.Commit.Author.Date,
			Author: argus.Author{
				Login: c.Author.Login,
				ID:    c.Author.ID,
				Name:  c.Commit.Author.Name,
				Email: c.Commit.Author.Email,
			},
		})
	}
	return ret
}

func (c *crawler) getLogger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"type": "github",
		"user": c.svc.User,
	})
}

// Generated struct from Github JSON
type (
	repo struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Owner struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"owner"`
		Private          bool        `json:"private"`
		HTMLURL          string      `json:"html_url"`
		Description      string      `json:"description"`
		Fork             bool        `json:"fork"`
		URL              string      `json:"url"`
		ForksURL         string      `json:"forks_url"`
		KeysURL          string      `json:"keys_url"`
		CollaboratorsURL string      `json:"collaborators_url"`
		TeamsURL         string      `json:"teams_url"`
		HooksURL         string      `json:"hooks_url"`
		IssueEventsURL   string      `json:"issue_events_url"`
		EventsURL        string      `json:"events_url"`
		AssigneesURL     string      `json:"assignees_url"`
		BranchesURL      string      `json:"branches_url"`
		TagsURL          string      `json:"tags_url"`
		BlobsURL         string      `json:"blobs_url"`
		GitTagsURL       string      `json:"git_tags_url"`
		GitRefsURL       string      `json:"git_refs_url"`
		TreesURL         string      `json:"trees_url"`
		StatusesURL      string      `json:"statuses_url"`
		LanguagesURL     string      `json:"languages_url"`
		StargazersURL    string      `json:"stargazers_url"`
		ContributorsURL  string      `json:"contributors_url"`
		SubscribersURL   string      `json:"subscribers_url"`
		SubscriptionURL  string      `json:"subscription_url"`
		CommitsURL       string      `json:"commits_url"`
		GitCommitsURL    string      `json:"git_commits_url"`
		CommentsURL      string      `json:"comments_url"`
		IssueCommentURL  string      `json:"issue_comment_url"`
		ContentsURL      string      `json:"contents_url"`
		CompareURL       string      `json:"compare_url"`
		MergesURL        string      `json:"merges_url"`
		ArchiveURL       string      `json:"archive_url"`
		DownloadsURL     string      `json:"downloads_url"`
		IssuesURL        string      `json:"issues_url"`
		PullsURL         string      `json:"pulls_url"`
		MilestonesURL    string      `json:"milestones_url"`
		NotificationsURL string      `json:"notifications_url"`
		LabelsURL        string      `json:"labels_url"`
		ReleasesURL      string      `json:"releases_url"`
		DeploymentsURL   string      `json:"deployments_url"`
		CreatedAt        time.Time   `json:"created_at"`
		UpdatedAt        time.Time   `json:"updated_at"`
		PushedAt         time.Time   `json:"pushed_at"`
		GitURL           string      `json:"git_url"`
		SSHURL           string      `json:"ssh_url"`
		CloneURL         string      `json:"clone_url"`
		SvnURL           string      `json:"svn_url"`
		Homepage         interface{} `json:"homepage"`
		Size             int         `json:"size"`
		StargazersCount  int         `json:"stargazers_count"`
		WatchersCount    int         `json:"watchers_count"`
		Language         string      `json:"language"`
		HasIssues        bool        `json:"has_issues"`
		HasProjects      bool        `json:"has_projects"`
		HasDownloads     bool        `json:"has_downloads"`
		HasWiki          bool        `json:"has_wiki"`
		HasPages         bool        `json:"has_pages"`
		ForksCount       int         `json:"forks_count"`
		MirrorURL        interface{} `json:"mirror_url"`
		Archived         bool        `json:"archived"`
		OpenIssuesCount  int         `json:"open_issues_count"`
		License struct {
			Key    string `json:"key"`
			Name   string `json:"name"`
			SpdxID string `json:"spdx_id"`
			URL    string `json:"url"`
		} `json:"license"`
		Forks         int    `json:"forks"`
		OpenIssues    int    `json:"open_issues"`
		Watchers      int    `json:"watchers"`
		DefaultBranch string `json:"default_branch"`
		Permissions struct {
			Admin bool `json:"admin"`
			Push  bool `json:"push"`
			Pull  bool `json:"pull"`
		} `json:"permissions"`
	}

	commit struct {
		Sha string `json:"sha"`
		Commit struct {
			Author struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"author"`
			Committer struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"committer"`
			Message string `json:"message"`
			Tree struct {
				Sha string `json:"sha"`
				URL string `json:"url"`
			} `json:"tree"`
			URL          string `json:"url"`
			CommentCount int    `json:"comment_count"`
			Verification struct {
				Verified  bool        `json:"verified"`
				Reason    string      `json:"reason"`
				Signature interface{} `json:"signature"`
				Payload   interface{} `json:"payload"`
			} `json:"verification"`
		} `json:"commit"`
		URL         string `json:"url"`
		HTMLURL     string `json:"html_url"`
		CommentsURL string `json:"comments_url"`
		Author struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"author"`
		Committer struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"committer"`
		Parents []struct {
			Sha     string `json:"sha"`
			URL     string `json:"url"`
			HTMLURL string `json:"html_url"`
		} `json:"parents"`
	}
)
