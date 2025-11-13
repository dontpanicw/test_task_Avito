package usecase

import (
	"context"
	entity2 "test_task_avito/backend/internal/entity"
	port2 "test_task_avito/backend/internal/port"
)

type userUseCase struct {
	userRepo port2.UserRepository
	prRepo   port2.PullRequestRepository
}

// NewUserUseCase создает новый экземпляр UserUseCase
func NewUserUseCase(userRepo port2.UserRepository, prRepo port2.PullRequestRepository) port2.UserUseCase {
	return &userUseCase{
		userRepo: userRepo,
		prRepo:   prRepo,
	}
}

func (uc *userUseCase) SetUserIsActive(ctx context.Context, userID string, isActive bool) (*entity2.User, error) {
	// Обновляем флаг активности
	if err := uc.userRepo.UpdateUserIsActive(ctx, userID, isActive); err != nil {
		return nil, err
	}

	// Получаем обновленного пользователя
	return uc.userRepo.GetUser(ctx, userID)
}

func (uc *userUseCase) GetUserReviews(ctx context.Context, userID string) ([]*entity2.PullRequest, error) {
	// Проверяем существование пользователя
	_, err := uc.userRepo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Получаем PR'ы, где пользователь назначен ревьювером
	return uc.prRepo.GetPullRequestsByReviewer(ctx, userID)
}
