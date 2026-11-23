package webhook

import (
	"encoding/json"
	"strings"
)

// Event represents a normalized Git webhook trigger event
type Event struct {
	Platform string `json:"platform"` // github, gitlab, generic
	Branch   string `json:"branch"`
	Commit   string `json:"commit"`
	Author   string `json:"author"`
	Message  string `json:"message"`
}

func getHeader(headers map[string]string, key string) string {
	for k, v := range headers {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}

// ParseEvent inspects HTTP headers and payload to normalize git event
func ParseEvent(headers map[string]string, payload []byte) (*Event, error) {
	// 1. GitHub
	if eventType := getHeader(headers, "X-GitHub-Event"); eventType != "" {
		var gh struct {
			Ref        string `json:"ref"` // refs/heads/main
			HeadCommit struct {
				ID      string `json:"id"`
				Message string `json:"message"`
				Author  struct {
					Name string `json:"name"`
				} `json:"author"`
			} `json:"head_commit"`
		}
		_ = json.Unmarshal(payload, &gh)
		branch := strings.TrimPrefix(gh.Ref, "refs/heads/")
		return &Event{
			Platform: "github",
			Branch:   branch,
			Commit:   gh.HeadCommit.ID,
			Author:   gh.HeadCommit.Author.Name,
			Message:  gh.HeadCommit.Message,
		}, nil
	}

	// 2. GitLab
	if eventType := getHeader(headers, "X-Gitlab-Event"); eventType != "" {
		var gl struct {
			Ref      string `json:"ref"`
			UserName string `json:"user_name"`
			Commits  []struct {
				ID      string `json:"id"`
				Message string `json:"message"`
			} `json:"commits"`
		}
		_ = json.Unmarshal(payload, &gl)
		branch := strings.TrimPrefix(gl.Ref, "refs/heads/")
		commitID := ""
		msg := ""
		if len(gl.Commits) > 0 {
			commitID = gl.Commits[0].ID
			msg = gl.Commits[0].Message
		}
		return &Event{
			Platform: "gitlab",
			Branch:   branch,
			Commit:   commitID,
			Author:   gl.UserName,
			Message:  msg,
		}, nil
	}

	// 3. Generic fallback
	var generic struct {
		Branch  string `json:"branch"`
		Commit  string `json:"commit"`
		Author  string `json:"author"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(payload, &generic); err == nil && generic.Branch != "" {
		return &Event{
			Platform: "generic",
			Branch:   generic.Branch,
			Commit:   generic.Commit,
			Author:   generic.Author,
			Message:  generic.Message,
		}, nil
	}

	return &Event{
		Platform: "generic",
		Branch:   "main",
		Author:   "webhook",
	}, nil
}
