package lib

type (
	SVC struct {
		User string `json:"username"`
		APIKey string `json:"api_key"`
	}

	Day struct {
		Private uint `json:"private"`
		Public uint `json:"public"`
	}

	Crawler interface {
		GetContributions() []*Day
	}
)
