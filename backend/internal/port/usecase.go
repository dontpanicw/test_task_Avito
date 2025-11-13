package port

import (
	"context"
	entity2 "test_task_avito/backend/internal/entity"
)

// TeamUseCase интерфейс для бизнес-логики команд
type TeamUseCase interface {
	// CreateTeam создает команду с участниками (создает/обновляет пользователей)
	CreateTeam(ctx context.Context, team *entity2.Team) error
	// GetTeam получает команду с участниками
	GetTeam(ctx context.Context, teamName string) (*entity2.Team, error)
}

// UserUseCase интерфейс для бизнес-логики пользователей
type UserUseCase interface {
	// SetUserIsActive устанавливает флаг активности пользователя
	SetUserIsActive(ctx context.Context, userID string, isActive bool) (*entity2.User, error)
	// GetUserReviews получает PR'ы, где пользователь назначен ревьювером
	GetUserReviews(ctx context.Context, userID string) ([]*entity2.PullRequest, error)
}

// PullRequestUseCase интерфейс для бизнес-логики Pull Request'ов
type PullRequestUseCase interface {
	// CreatePullRequest создает PR и автоматически назначает до 2 ревьюверов из команды автора
	CreatePullRequest(ctx context.Context, prID, prName, authorID string) (*entity2.PullRequest, error)
	// MergePullRequest помечает PR как MERGED (идемпотентная операция)
	MergePullRequest(ctx context.Context, prID string) (*entity2.PullRequest, error)
	// ReassignReviewer переназначает конкретного ревьювера на другого из его команды
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (*entity2.PullRequest, string, error)
}
