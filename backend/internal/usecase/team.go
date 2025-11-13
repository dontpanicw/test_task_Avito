package usecase

import (
	"context"
	entity2 "test_task_avito/backend/internal/entity"
	port2 "test_task_avito/backend/internal/port"
)

type teamUseCase struct {
	teamRepo port2.TeamRepository
	userRepo port2.UserRepository
}

// NewTeamUseCase создает новый экземпляр TeamUseCase
func NewTeamUseCase(teamRepo port2.TeamRepository, userRepo port2.UserRepository) port2.TeamUseCase {
	return &teamUseCase{
		teamRepo: teamRepo,
		userRepo: userRepo,
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
