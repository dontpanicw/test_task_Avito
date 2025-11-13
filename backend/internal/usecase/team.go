package usecase

import (
	"context"
	"errors"
	"math/rand"
	entity2 "test_task_avito/backend/internal/entity"
	port2 "test_task_avito/backend/internal/port"
	"time"
)

type teamUseCase struct {
	teamRepo port2.TeamRepository
	userRepo port2.UserRepository
	prRepo   port2.PullRequestRepository
	rng      *rand.Rand
}

// NewTeamUseCase создает новый экземпляр TeamUseCase
func NewTeamUseCase(teamRepo port2.TeamRepository, userRepo port2.UserRepository, prRepo port2.PullRequestRepository) port2.TeamUseCase {
	return &teamUseCase{
		teamRepo: teamRepo,
		userRepo: userRepo,
		prRepo:   prRepo,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (uc *teamUseCase) CreateTeam(ctx context.Context, team *entity2.Team) error {
	// Создаем команду (репозиторий проверит существование)
	if err := uc.teamRepo.CreateTeam(ctx, team); err != nil {
		return err
	}

	// Создаем/обновляем пользователей
	for _, member := range team.Members {
		user := &entity2.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: team.TeamName,
			IsActive: member.IsActive,
		}
		if err := uc.userRepo.CreateOrUpdateUser(ctx, user); err != nil {
			return err
		}
	}

	return nil
}

func (uc *teamUseCase) GetTeam(ctx context.Context, teamName string) (*entity2.Team, error) {
	return uc.teamRepo.GetTeam(ctx, teamName)
}

func (uc *teamUseCase) DeactivateTeam(ctx context.Context, teamName string, strategy entity2.ReplacementStrategy) (*entity2.TeamDeactivateResult, error) {
	strategy = strategy.Normalize()
	if !strategy.Valid() {
		return nil, entity2.NewDomainError(entity2.ErrorCodeNoCandidate, "invalid replacement strategy")
	}

	exists, err := uc.teamRepo.TeamExists(ctx, teamName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, entity2.NewDomainError(entity2.ErrorCodeNotFound, "team not found")
	}

	users, err := uc.userRepo.GetUsersByTeam(ctx, teamName)
	if err != nil {
		return nil, err
	}

	activeUsers := make([]*entity2.User, 0, len(users))
	targetSet := make(map[string]struct{}, len(users))
	for _, user := range users {
		if user.IsActive {
			activeUsers = append(activeUsers, user)
			targetSet[user.UserID] = struct{}{}
		}
	}

	if len(activeUsers) == 0 {
		return nil, entity2.NewDomainError(entity2.ErrorCodeNotFound, "team has no active users to deactivate")
	}

	reviewerIDs := make([]string, 0, len(activeUsers))
	userByID := make(map[string]*entity2.User, len(activeUsers))
	for _, user := range activeUsers {
		reviewerIDs = append(reviewerIDs, user.UserID)
		userByID[user.UserID] = user
	}

	prIDs, err := uc.prRepo.GetOpenPullRequestsByReviewers(ctx, reviewerIDs)
	if err != nil {
		return nil, err
	}

	result := &entity2.TeamDeactivateResult{
		TeamName: teamName,
	}

	authorCache := make(map[string]*entity2.User)
	skippedPRs := make(map[string]struct{})

	for _, prID := range prIDs {
		pr, err := uc.prRepo.GetPullRequest(ctx, prID)
		if err != nil {
			return nil, err
		}
		if pr.Status != entity2.PullRequestStatusOpen {
			continue
		}

		currentReviewers := make(map[string]struct{}, len(pr.AssignedReviewers))
		for _, reviewer := range pr.AssignedReviewers {
			currentReviewers[reviewer] = struct{}{}
		}

		newReviewers := make([]string, 0, len(pr.AssignedReviewers))
		prSkipped := false

		for _, reviewer := range pr.AssignedReviewers {
			if _, toDeactivate := targetSet[reviewer]; !toDeactivate {
				newReviewers = append(newReviewers, reviewer)
				continue
			}

			delete(currentReviewers, reviewer)

			replacement, repErr := uc.pickReplacement(ctx, strategy, reviewer, pr, currentReviewers, targetSet, userByID, authorCache)
			if repErr != nil {
				if errors.Is(repErr, errNoReplacement) {
					prSkipped = true
					continue
				}
				return nil, repErr
			}

			newReviewers = append(newReviewers, replacement)
			currentReviewers[replacement] = struct{}{}
			result.ReassignedPRs++
		}

		if prSkipped {
			skippedPRs[prID] = struct{}{}
		}

		if err := uc.prRepo.UpdatePullRequestReviewers(ctx, prID, newReviewers); err != nil {
			return nil, err
		}
	}

	deactivatedCount, err := uc.teamRepo.BulkDeactivateUsersByTeam(ctx, teamName)
	if err != nil {
		return nil, err
	}
	result.DeactivatedUsers = deactivatedCount
	result.SkippedPRs = int64(len(skippedPRs))

	if result.SkippedPRs > 0 {
		return result, entity2.NewDomainError(entity2.ErrorCodeNoCandidate, "unable to reassign all reviewers")
	}

	return result, nil
}

var errNoReplacement = errors.New("no replacement found")

func (uc *teamUseCase) pickReplacement(
	ctx context.Context,
	strategy entity2.ReplacementStrategy,
	reviewerID string,
	pr *entity2.PullRequest,
	currentReviewers map[string]struct{},
	toDeactivate map[string]struct{},
	userByID map[string]*entity2.User,
	authorCache map[string]*entity2.User,
) (string, error) {

	oldUser, ok := userByID[reviewerID]
	if !ok {
		return "", entity2.NewDomainError(entity2.ErrorCodeNotFound, "reviewer not found for reassignment")
	}

	candidates := make([]*entity2.User, 0)

	if strategy == entity2.ReplacementStrategySameTeam {
		sameTeamCandidates, err := uc.userRepo.GetActiveUsersByTeam(ctx, oldUser.TeamName, reviewerID)
		if err != nil {
			return "", err
		}
		candidates = append(candidates, uc.filterCandidates(sameTeamCandidates, currentReviewers, toDeactivate)...)
	}

	if strategy == entity2.ReplacementStrategyAuthorTeam || len(candidates) == 0 {
		author, err := uc.getAuthor(ctx, pr.AuthorID, authorCache)
		if err != nil {
			return "", err
		}
		if author != nil {
			authorCandidates, err := uc.userRepo.GetActiveUsersByTeam(ctx, author.TeamName, author.UserID)
			if err != nil {
				return "", err
			}
			candidates = append(candidates, uc.filterCandidates(authorCandidates, currentReviewers, toDeactivate)...)
		}
	}

	if len(candidates) == 0 {
		exclude := make([]string, 0, len(currentReviewers)+len(toDeactivate)+1)
		for id := range currentReviewers {
			exclude = append(exclude, id)
		}
		for id := range toDeactivate {
			exclude = append(exclude, id)
		}
		exclude = append(exclude, reviewerID)

		globalCandidates, err := uc.userRepo.GetAllActiveUsers(ctx, exclude)
		if err != nil {
			return "", err
		}
		candidates = append(candidates, uc.filterCandidates(globalCandidates, currentReviewers, toDeactivate)...)
	}

	if len(candidates) == 0 {
		return "", errNoReplacement
	}

	uc.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	return candidates[0].UserID, nil
}

func (uc *teamUseCase) filterCandidates(candidates []*entity2.User, currentReviewers map[string]struct{}, toDeactivate map[string]struct{}) []*entity2.User {
	filtered := make([]*entity2.User, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))

	for _, candidate := range candidates {
		if _, already := currentReviewers[candidate.UserID]; already {
			continue
		}
		if _, deactivated := toDeactivate[candidate.UserID]; deactivated {
			continue
		}
		if _, exists := seen[candidate.UserID]; exists {
			continue
		}
		seen[candidate.UserID] = struct{}{}
		filtered = append(filtered, candidate)
	}

	return filtered
}

func (uc *teamUseCase) getAuthor(ctx context.Context, authorID string, cache map[string]*entity2.User) (*entity2.User, error) {
	if author, ok := cache[authorID]; ok {
		return author, nil
	}
	author, err := uc.userRepo.GetUser(ctx, authorID)
	if err != nil {
		if domainErr, ok := err.(*entity2.DomainError); ok && domainErr.Code == entity2.ErrorCodeNotFound {
			cache[authorID] = nil
			return nil, nil
		}
		return nil, err
	}
	cache[authorID] = author
	return author, nil
}
