package entity

import "time"

// PullRequestStatus представляет статус PR
type PullRequestStatus string

const (
	PullRequestStatusOpen   PullRequestStatus = "OPEN"
	PullRequestStatusMerged PullRequestStatus = "MERGED"
)

// PullRequest представляет Pull Request
type PullRequest struct {
	PullRequestID     string
	PullRequestName   string
	AuthorID          string
	Status            PullRequestStatus
	AssignedReviewers []string // user_id назначенных ревьюверов (0..2)
	CreatedAt         *time.Time
	MergedAt          *time.Time
}

