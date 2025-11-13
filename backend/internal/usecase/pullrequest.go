package usecase

import (
	"context"
	"math/rand"
	entity2 "test_task_avito/backend/internal/entity"
	port2 "test_task_avito/backend/internal/port"
	"time"
)

type pullRequestUseCase struct {
	prRepo   port2.PullRequestRepository
	userRepo port2.UserRepository
	teamRepo port2.TeamRepository
}

// NewPullRequestUseCase создает новый экземпляр PullRequestUseCase
func NewPullRequestUseCase(prRepo port2.PullRequestRepository, userRepo port2.UserRepository, teamRepo port2.TeamRepository) port2.PullRequestUseCase {
	return &pullRequestUseCase{
		prRepo:   prRepo,
		userRepo: userRepo,
		teamRepo: teamRepo,
	}
}

func (uc *pullRequestUseCase) CreatePullRequest(ctx context.Context, prID, prName, authorID string) (*entity2.PullRequest, error) {
	// Проверяем существование PR
	exists, err := uc.prRepo.PRExists(ctx, prID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, entity2.NewDomainError(entity2.ErrorCodePRExists, "PR id already exists")
	}

	// Получаем автора
	author, err := uc.userRepo.GetUser(ctx, authorID)
	if err != nil {
		return nil, entity2.NewDomainError(entity2.ErrorCodeNotFound, "author not found")
	}

	// Получаем активных пользователей команды автора (исключая самого автора)
	candidates, err := uc.userRepo.GetActiveUsersByTeam(ctx, author.TeamName, authorID)
	if err != nil {
		return nil, err
	}

	// Назначаем до 2 ревьюверов
	reviewers := uc.selectReviewers(candidates, 2)

	now := time.Now()
	pr := &entity2.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   prName,
		AuthorID:          authorID,
		Status:            entity2.PullRequestStatusOpen,
		AssignedReviewers: reviewers,
		CreatedAt:         &now,
	}

	if err := uc.prRepo.CreatePullRequest(ctx, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (uc *pullRequestUseCase) MergePullRequest(ctx context.Context, prID string) (*entity2.PullRequest, error) {
	// Получаем PR
	pr, err := uc.prRepo.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, err
	}

	// Если уже MERGED, возвращаем текущее состояние (идемпотентность)
	if pr.Status == entity2.PullRequestStatusMerged {
		return pr, nil
	}

	// Обновляем статус
	now := time.Now()
	if err := uc.prRepo.UpdatePullRequestStatus(ctx, prID, entity2.PullRequestStatusMerged, &now); err != nil {
		return nil, err
	}

	pr.Status = entity2.PullRequestStatusMerged
	pr.MergedAt = &now

	return pr, nil
}

func (uc *pullRequestUseCase) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*entity2.PullRequest, string, error) {
	// Получаем PR
	pr, err := uc.prRepo.GetPullRequest(ctx, prID)
	if err != nil {
		return nil, "", err
	}

	// Проверяем, что PR не MERGED
	if pr.Status == entity2.PullRequestStatusMerged {
		return nil, "", entity2.NewDomainError(entity2.ErrorCodePRMerged, "cannot reassign on merged PR")
	}

	// Проверяем, что oldUserID назначен ревьювером
	found := false
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID == oldUserID {
			found = true
			break
		}
	}
	if !found {
		return nil, "", entity2.NewDomainError(entity2.ErrorCodeNotAssigned, "reviewer is not assigned to this PR")
	}

	// Получаем пользователя, которого заменяем
	oldUser, err := uc.userRepo.GetUser(ctx, oldUserID)
	if err != nil {
		return nil, "", err
	}

	// Получаем активных пользователей команды заменяемого ревьювера (исключая его самого и автора PR)
	candidates, err := uc.userRepo.GetActiveUsersByTeam(ctx, oldUser.TeamName, oldUserID)
	if err != nil {
		return nil, "", err
	}

	// Исключаем автора PR из кандидатов
	var filteredCandidates []*entity2.User
	for _, candidate := range candidates {
		if candidate.UserID != pr.AuthorID {
			filteredCandidates = append(filteredCandidates, candidate)
		}
	}

	// Исключаем уже назначенных ревьюверов
	var availableCandidates []*entity2.User
	for _, candidate := range filteredCandidates {
		isAssigned := false
		for _, reviewerID := range pr.AssignedReviewers {
			if candidate.UserID == reviewerID {
				isAssigned = true
				break
			}
		}
		if !isAssigned {
			availableCandidates = append(availableCandidates, candidate)
		}
	}

	if len(availableCandidates) == 0 {
		return nil, "", entity2.NewDomainError(entity2.ErrorCodeNoCandidate, "no active replacement candidate in team")
	}

	// Выбираем случайного кандидата
	newReviewer := availableCandidates[rand.Intn(len(availableCandidates))]

	// Обновляем список ревьюверов
	newReviewers := make([]string, 0, len(pr.AssignedReviewers))
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID != oldUserID {
			newReviewers = append(newReviewers, reviewerID)
		}
	}
	newReviewers = append(newReviewers, newReviewer.UserID)

	if err := uc.prRepo.UpdatePullRequestReviewers(ctx, prID, newReviewers); err != nil {
		return nil, "", err
	}

	pr.AssignedReviewers = newReviewers

	return pr, newReviewer.UserID, nil
}

// selectReviewers выбирает до maxReviewers случайных ревьюверов из кандидатов
func (uc *pullRequestUseCase) selectReviewers(candidates []*entity2.User, maxReviewers int) []string {
	if len(candidates) == 0 {
		return []string{}
	}

	count := maxReviewers
	if len(candidates) < count {
		count = len(candidates)
	}

	// Перемешиваем кандидатов
	shuffled := make([]*entity2.User, len(candidates))
	copy(shuffled, candidates)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Выбираем первых count
	reviewers := make([]string, 0, count)
	for i := 0; i < count; i++ {
		reviewers = append(reviewers, shuffled[i].UserID)
	}

	return reviewers
}

func (uc *pullRequestUseCase) GetReviewerStats(ctx context.Context) ([]entity2.ReviewerStat, int64, error) {
	return uc.prRepo.GetReviewerStats(ctx)
}
