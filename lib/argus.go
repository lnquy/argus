package lib

import "time"

type (
	SVC struct {
		User   string   `json:"username"`
		Emails  []string `json:"emails"`
		APIKey string   `json:"api_key"`
	}

	Repo struct {
		Name      string    `json:"name"`
		FullName  string    `json:"full_name"`
		URL       string    `json:"url"`
		Private   bool      `json:"private"`
		CreatedAt time.Time `json:"created_at"`
		PushedAt  time.Time `json:"pushed_at"`
		Commits   []Commit  `json:"commits"`
	}

	Commit struct {
		Sha    string    `json:"sha"`
		Date   time.Time `json:"date"`
		Author Author    `json:"author"`
	}

	Author struct {
		Login string `json:"login"`
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	Crawler interface {
		GetContributions() ([]Repo, error)
	}
)
