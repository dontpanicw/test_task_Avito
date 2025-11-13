package port

import (
	"context"
	entity2 "test_task_avito/backend/internal/entity"
	"time"
)

// TeamRepository интерфейс для работы с командами
type TeamRepository interface {
	// CreateTeam создает команду (если команда существует, возвращает ошибку)
	CreateTeam(ctx context.Context, team *entity2.Team) error
	// GetTeam получает команду по имени
	GetTeam(ctx context.Context, teamName string) (*entity2.Team, error)
	// TeamExists проверяет существование команды
	TeamExists(ctx context.Context, teamName string) (bool, error)
	// BulkDeactivateUsersByTeam деактивирует пользователей команды и возвращает количество обновленных записей
	BulkDeactivateUsersByTeam(ctx context.Context, teamName string) (int64, error)
}

// UserRepository интерфейс для работы с пользователями
type UserRepository interface {
	// CreateOrUpdateUser создает или обновляет пользователя
	CreateOrUpdateUser(ctx context.Context, user *entity2.User) error
	// GetUser получает пользователя по ID
	GetUser(ctx context.Context, userID string) (*entity2.User, error)
	// GetActiveUsersByTeam получает активных пользователей команды (исключая указанного)
	GetActiveUsersByTeam(ctx context.Context, teamName string, excludeUserID string) ([]*entity2.User, error)
	// UpdateUserIsActive обновляет флаг активности пользователя
	UpdateUserIsActive(ctx context.Context, userID string, isActive bool) error
	// GetUsersByTeam получает всех пользователей команды (включая неактивных)
	GetUsersByTeam(ctx context.Context, teamName string) ([]*entity2.User, error)
	// GetAllActiveUsers возвращает всех активных пользователей с возможностью исключения
	GetAllActiveUsers(ctx context.Context, excludeIDs []string) ([]*entity2.User, error)
}

// PullRequestRepository интерфейс для работы с Pull Request'ами
type PullRequestRepository interface {
	// CreatePullRequest создает PR
	CreatePullRequest(ctx context.Context, pr *entity2.PullRequest) error
	// GetPullRequest получает PR по ID
	GetPullRequest(ctx context.Context, prID string) (*entity2.PullRequest, error)
	// PRExists проверяет существование PR
	PRExists(ctx context.Context, prID string) (bool, error)
	// UpdatePullRequestStatus обновляет статус PR
	UpdatePullRequestStatus(ctx context.Context, prID string, status entity2.PullRequestStatus, mergedAt *time.Time) error
	// UpdatePullRequestReviewers обновляет список ревьюверов PR
	UpdatePullRequestReviewers(ctx context.Context, prID string, reviewers []string) error
	// GetPullRequestsByReviewer получает PR'ы, где пользователь назначен ревьювером
	GetPullRequestsByReviewer(ctx context.Context, userID string) ([]*entity2.PullRequest, error)
	// GetOpenPullRequestsByReviewers возвращает ID открытых PR, где задействованы ревьюверы из списка
	GetOpenPullRequestsByReviewers(ctx context.Context, reviewerIDs []string) ([]string, error)
	// GetReviewerStats возвращает статистику по назначенным ревьюверам
	GetReviewerStats(ctx context.Context) ([]entity2.ReviewerStat, int64, error)
}
