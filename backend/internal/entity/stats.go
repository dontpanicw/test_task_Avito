package entity

type ReviewerStat struct {
	UserID       string
	ReviewsCount int64
}

type ReplacementStrategy string

const (
	ReplacementStrategySameTeam   ReplacementStrategy = "same_team"
	ReplacementStrategyAuthorTeam ReplacementStrategy = "author_team"
)

type TeamDeactivateResult struct {
	TeamName         string
	DeactivatedUsers int64
	ReassignedPRs    int64
	SkippedPRs       int64
}

func (s ReplacementStrategy) Valid() bool {
	return s == ReplacementStrategySameTeam || s == ReplacementStrategyAuthorTeam || s == ""
}

func (s ReplacementStrategy) Normalize() ReplacementStrategy {
	if s == "" {
		return ReplacementStrategySameTeam
	}
	return s
}
