package integration_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"test_task_avito/backend/internal/adapter/repository/postgres"
	"test_task_avito/backend/internal/input/http/gen"
	handlerpkg "test_task_avito/backend/internal/input/http/handler"
	"test_task_avito/backend/internal/usecase"
	migrations "test_task_avito/backend/pkg/migration"
)

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type teamDeactivateResult struct {
	TeamName    string `json:"team_name"`
	Deactivated int64  `json:"deactivated_users"`
	Reassigned  int64  `json:"reassigned_prs"`
	Skipped     int64  `json:"skipped_prs"`
}

type reviewerStats struct {
	Stats []struct {
		UserID       string `json:"user_id"`
		ReviewsCount int64  `json:"reviews_count"`
	} `json:"stats"`
	Total int64 `json:"total_reviews"`
}

func TestEndToEndFlow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker binary not found; skipping integration test")
	}

	pgContainer, err := tcpostgres.RunContainer(ctx,
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("secret"),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.Eventually(t, func() bool {
		ctxPing, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return db.PingContext(ctxPing) == nil
	}, 30*time.Second, 1*time.Second)

	require.NoError(t, migrations.Migrate(db))

	repo := postgres.NewPostgresRepository(db)
	teamUC := usecase.NewTeamUseCase(repo, repo, repo)
	userUC := usecase.NewUserUseCase(repo, repo)
	prUC := usecase.NewPullRequestUseCase(repo, repo, repo)

	h := handlerpkg.NewHandler(teamUC, userUC, prUC)
	strictHandler := gen.NewStrictHandler(h, nil)

	r := chi.NewRouter()
	gen.HandlerFromMux(strictHandler, r)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	client := &http.Client{Timeout: 5 * time.Second}

	mustDo(t, client, srv, http.MethodPost, "/team/add", map[string]any{
		"team_name": "backend",
		"members": []map[string]any{
			{"user_id": "u1", "username": "User1", "is_active": true},
			{"user_id": "u2", "username": "User2", "is_active": true},
			{"user_id": "u3", "username": "User3", "is_active": true},
		},
	}, http.StatusCreated)

	mustDo(t, client, srv, http.MethodPost, "/team/add", map[string]any{
		"team_name": "platform",
		"members": []map[string]any{
			{"user_id": "p1", "username": "Platform1", "is_active": true},
			{"user_id": "p2", "username": "Platform2", "is_active": true},
		},
	}, http.StatusCreated)

	mustDo(t, client, srv, http.MethodPost, "/pullRequest/create", map[string]any{
		"pull_request_id":   "pr-1",
		"pull_request_name": "Feature",
		"author_id":         "u1",
	}, http.StatusCreated)

	resp := mustDo(t, client, srv, http.MethodGet, "/stats/reviewers", nil, http.StatusOK)
	var stats reviewerStats
	decodeJSON(t, resp.Body, &stats)
	require.Greater(t, stats.Total, int64(0))

	deactivateResp := mustDo(t, client, srv, http.MethodPost, "/team/deactivate", map[string]any{
		"team_name":            "backend",
		"replacement_strategy": "author_team",
	}, http.StatusOK)
	var deactivateResult teamDeactivateResult
	decodeJSON(t, deactivateResp.Body, &deactivateResult)
	require.Equal(t, "backend", deactivateResult.TeamName)
	require.Greater(t, deactivateResult.Deactivated, int64(0))
}

func mustDo(t *testing.T, client *http.Client, srv *httptest.Server, method, path string, body any, expected int) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, srv.URL+path, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })

	require.Equal(t, expected, resp.StatusCode, "unexpected status for %s %s", method, path)
	return resp
}

func decodeJSON(t *testing.T, r io.Reader, out any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(r).Decode(out))
}
