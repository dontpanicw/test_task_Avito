package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	entity2 "test_task_avito/backend/internal/entity"
	"test_task_avito/backend/internal/port"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Проверка реализации интерфейсов
var (
	_ port.TeamRepository        = (*PostgresRepository)(nil)
	_ port.UserRepository        = (*PostgresRepository)(nil)
	_ port.PullRequestRepository = (*PostgresRepository)(nil)
)

// PostgresRepository объединяет все репозитории
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создает новый экземпляр PostgresRepository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// TeamRepository реализация
func (r *PostgresRepository) CreateTeam(ctx context.Context, team *entity2.Team) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Проверяем, существует ли команда
	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", team.TeamName).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return entity2.NewDomainError(entity2.ErrorCodeTeamExists, "team_name already exists")
	}

	// Создаем команду
	_, err = tx.ExecContext(ctx, "INSERT INTO teams (team_name) VALUES ($1)", team.TeamName)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetTeam(ctx context.Context, teamName string) (*entity2.Team, error) {
	// Проверяем существование команды
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, entity2.NewDomainError(entity2.ErrorCodeNotFound, "team not found")
	}

	// Получаем всех пользователей команды
	rows, err := r.db.QueryContext(ctx,
		"SELECT user_id, username, is_active FROM users WHERE team_name = $1 ORDER BY user_id",
		teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []entity2.TeamMember
	for rows.Next() {
		var member entity2.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &entity2.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}

func (r *PostgresRepository) TeamExists(ctx context.Context, teamName string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) BulkDeactivateUsersByTeam(ctx context.Context, teamName string) (int64, error) {
	result, err := r.db.ExecContext(ctx, "UPDATE users SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE team_name = $1 AND is_active = true", teamName)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// UserRepository реализация
func (r *PostgresRepository) CreateOrUpdateUser(ctx context.Context, user *entity2.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (user_id, username, team_name, is_active, updated_at)
		 VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		 ON CONFLICT (user_id) 
		 DO UPDATE SET username = EXCLUDED.username, team_name = EXCLUDED.team_name, 
		               is_active = EXCLUDED.is_active, updated_at = CURRENT_TIMESTAMP`,
		user.UserID, user.Username, user.TeamName, user.IsActive)
	return err
}

func (r *PostgresRepository) GetUser(ctx context.Context, userID string) (*entity2.User, error) {
	var user entity2.User
	err := r.db.QueryRowContext(ctx,
		"SELECT user_id, username, team_name, is_active FROM users WHERE user_id = $1",
		userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
	if err == sql.ErrNoRows {
		return nil, entity2.NewDomainError(entity2.ErrorCodeNotFound, "user not found")
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PostgresRepository) GetActiveUsersByTeam(ctx context.Context, teamName string, excludeUserID string) ([]*entity2.User, error) {
	query := "SELECT user_id, username, team_name, is_active FROM users WHERE team_name = $1 AND is_active = true"
	args := []interface{}{teamName}

	if excludeUserID != "" {
		query += " AND user_id != $2"
		args = append(args, excludeUserID)
	}

	query += " ORDER BY user_id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*entity2.User
	for rows.Next() {
		var user entity2.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}

func (r *PostgresRepository) UpdateUserIsActive(ctx context.Context, userID string, isActive bool) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE users SET is_active = $1, updated_at = CURRENT_TIMESTAMP WHERE user_id = $2",
		isActive, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return entity2.NewDomainError(entity2.ErrorCodeNotFound, "user not found")
	}

	return nil
}

func (r *PostgresRepository) GetUsersByTeam(ctx context.Context, teamName string) ([]*entity2.User, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT user_id, username, team_name, is_active FROM users WHERE team_name = $1 ORDER BY user_id",
		teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*entity2.User
	for rows.Next() {
		var user entity2.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}

func (r *PostgresRepository) GetAllActiveUsers(ctx context.Context, excludeIDs []string) ([]*entity2.User, error) {
	query := "SELECT user_id, username, team_name, is_active FROM users WHERE is_active = true"
	var args []interface{}

	if len(excludeIDs) > 0 {
		placeholders := make([]string, len(excludeIDs))
		args = make([]interface{}, len(excludeIDs))
		for i, id := range excludeIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args[i] = id
		}
		query += fmt.Sprintf(" AND user_id NOT IN (%s)", strings.Join(placeholders, ","))
	}

	query += " ORDER BY user_id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*entity2.User
	for rows.Next() {
		var user entity2.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}

// PullRequestRepository реализация
func (r *PostgresRepository) CreatePullRequest(ctx context.Context, pr *entity2.PullRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status, now)
	if err != nil {
		return err
	}

	// Добавляем ревьюверов
	for _, reviewerID := range pr.AssignedReviewers {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)",
			pr.PullRequestID, reviewerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetPullRequest(ctx context.Context, prID string) (*entity2.PullRequest, error) {
	var pr entity2.PullRequest
	var createdAt, mergedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		 FROM pull_requests WHERE pull_request_id = $1`,
		prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt)
	if err == sql.ErrNoRows {
		return nil, entity2.NewDomainError(entity2.ErrorCodeNotFound, "pull request not found")
	}
	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		pr.CreatedAt = &createdAt.Time
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	// Получаем ревьюверов
	rows, err := r.db.QueryContext(ctx,
		"SELECT reviewer_id FROM pull_request_reviewers WHERE pull_request_id = $1 ORDER BY reviewer_id",
		prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
	}

	return &pr, rows.Err()
}

func (r *PostgresRepository) PRExists(ctx context.Context, prID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)", prID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) UpdatePullRequestStatus(ctx context.Context, prID string, status entity2.PullRequestStatus, mergedAt *time.Time) error {
	var err error
	if mergedAt != nil {
		_, err = r.db.ExecContext(ctx,
			"UPDATE pull_requests SET status = $1, merged_at = $2 WHERE pull_request_id = $3",
			status, mergedAt, prID)
	} else {
		_, err = r.db.ExecContext(ctx,
			"UPDATE pull_requests SET status = $1 WHERE pull_request_id = $2",
			status, prID)
	}
	return err
}

func (r *PostgresRepository) UpdatePullRequestReviewers(ctx context.Context, prID string, reviewers []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Удаляем старых ревьюверов
	_, err = tx.ExecContext(ctx, "DELETE FROM pull_request_reviewers WHERE pull_request_id = $1", prID)
	if err != nil {
		return err
	}

	// Добавляем новых ревьюверов
	for _, reviewerID := range reviewers {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)",
			prID, reviewerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetPullRequestsByReviewer(ctx context.Context, userID string) ([]*entity2.PullRequest, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at
		 FROM pull_requests pr
		 INNER JOIN pull_request_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		 WHERE prr.reviewer_id = $1
		 ORDER BY pr.created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []*entity2.PullRequest
	for rows.Next() {
		var pr entity2.PullRequest
		var createdAt, mergedAt sql.NullTime

		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt); err != nil {
			return nil, err
		}

		if createdAt.Valid {
			pr.CreatedAt = &createdAt.Time
		}
		if mergedAt.Valid {
			pr.MergedAt = &mergedAt.Time
		}

		prs = append(prs, &pr)
	}

	return prs, rows.Err()
}

func (r *PostgresRepository) GetOpenPullRequestsByReviewers(ctx context.Context, reviewerIDs []string) ([]string, error) {
	if len(reviewerIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(reviewerIDs))
	args := make([]interface{}, len(reviewerIDs))
	for i, id := range reviewerIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`SELECT DISTINCT pr.pull_request_id
		FROM pull_requests pr
		INNER JOIN pull_request_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE pr.status = 'OPEN' AND prr.reviewer_id IN (%s)`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prIDs []string
	for rows.Next() {
		var prID string
		if err := rows.Scan(&prID); err != nil {
			return nil, err
		}
		prIDs = append(prIDs, prID)
	}

	return prIDs, rows.Err()
}

func (r *PostgresRepository) GetReviewerStats(ctx context.Context) ([]entity2.ReviewerStat, int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT reviewer_id, COUNT(*) AS reviews_count
		FROM pull_request_reviewers
		GROUP BY reviewer_id
		ORDER BY reviews_count DESC`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var stats []entity2.ReviewerStat
	var total int64

	for rows.Next() {
		var stat entity2.ReviewerStat
		if err := rows.Scan(&stat.UserID, &stat.ReviewsCount); err != nil {
			return nil, 0, err
		}
		stats = append(stats, stat)
		total += stat.ReviewsCount
	}

	return stats, total, rows.Err()
}
