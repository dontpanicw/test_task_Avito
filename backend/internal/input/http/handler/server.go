package handler

import (
	"context"
	entity2 "test_task_avito/backend/internal/entity"
	gen2 "test_task_avito/backend/internal/input/http/gen"
	"test_task_avito/backend/internal/port"
)

var _ gen2.StrictServerInterface = (*Handler)(nil)

type Handler struct {
	teamUseCase        port.TeamUseCase
	userUseCase        port.UserUseCase
	pullRequestUseCase port.PullRequestUseCase
}

func NewHandler(teamUseCase port.TeamUseCase, userUseCase port.UserUseCase, pullRequestUseCase port.PullRequestUseCase) *Handler {
	return &Handler{
		teamUseCase:        teamUseCase,
		userUseCase:        userUseCase,
		pullRequestUseCase: pullRequestUseCase,
	}
}

func (h *Handler) PostTeamAdd(ctx context.Context, request gen2.PostTeamAddRequestObject) (gen2.PostTeamAddResponseObject, error) {
	if request.Body == nil {
		return gen2.PostTeamAdd400JSONResponse{
			Error: struct {
				Code    gen2.ErrorResponseErrorCode `json:"code"`
				Message string                      `json:"message"`
			}{
				Code:    gen2.NOTFOUND,
				Message: "request body is required",
			},
		}, nil
	}

	// Конвертируем из gen в entity
	team := &entity2.Team{
		TeamName: request.Body.TeamName,
		Members:  make([]entity2.TeamMember, 0, len(request.Body.Members)),
	}

	for _, member := range request.Body.Members {
		team.Members = append(team.Members, entity2.TeamMember{
			UserID:   member.UserId,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}

	// Создаем команду
	err := h.teamUseCase.CreateTeam(ctx, team)
	if err != nil {
		if domainErr, ok := err.(*entity2.DomainError); ok {
			switch domainErr.Code {
			case entity2.ErrorCodeTeamExists:
				return gen2.PostTeamAdd400JSONResponse{
					Error: struct {
						Code    gen2.ErrorResponseErrorCode `json:"code"`
						Message string                      `json:"message"`
					}{
						Code:    gen2.TEAMEXISTS,
						Message: domainErr.Message,
					},
				}, nil
			}
		}
		return nil, err
	}

	// Конвертируем обратно в gen
	genTeam := &gen2.Team{
		TeamName: team.TeamName,
		Members:  make([]gen2.TeamMember, 0, len(team.Members)),
	}

	for _, member := range team.Members {
		genTeam.Members = append(genTeam.Members, gen2.TeamMember{
			UserId:   member.UserID,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}

	return gen2.PostTeamAdd201JSONResponse{Team: genTeam}, nil
}

func (h *Handler) PostTeamDeactivate(ctx context.Context, request gen2.PostTeamDeactivateRequestObject) (gen2.PostTeamDeactivateResponseObject, error) {
	if request.Body == nil {
		return gen2.PostTeamDeactivate404JSONResponse{
			Error: struct {
				Code    gen2.ErrorResponseErrorCode `json:"code"`
				Message string                      `json:"message"`
			}{
				Code:    gen2.NOTFOUND,
				Message: "request body is required",
			},
		}, nil
	}

	var strategy entity2.ReplacementStrategy
	if request.Body.ReplacementStrategy != nil {
		strategy = entity2.ReplacementStrategy(*request.Body.ReplacementStrategy)
	}

	result, err := h.teamUseCase.DeactivateTeam(ctx, request.Body.TeamName, strategy)
	if err != nil {
		if domainErr, ok := err.(*entity2.DomainError); ok {
			switch domainErr.Code {
			case entity2.ErrorCodeNotFound:
				return gen2.PostTeamDeactivate404JSONResponse{
					Error: struct {
						Code    gen2.ErrorResponseErrorCode `json:"code"`
						Message string                      `json:"message"`
					}{
						Code:    gen2.NOTFOUND,
						Message: domainErr.Message,
					},
				}, nil
			case entity2.ErrorCodeNoCandidate:
				return gen2.PostTeamDeactivate409JSONResponse{
					Error: struct {
						Code    gen2.ErrorResponseErrorCode `json:"code"`
						Message string                      `json:"message"`
					}{
						Code:    gen2.NOCANDIDATE,
						Message: domainErr.Message,
					},
				}, nil
			}
		}
		return nil, err
	}

	return gen2.PostTeamDeactivate200JSONResponse{
		TeamName:         result.TeamName,
		DeactivatedUsers: result.DeactivatedUsers,
		ReassignedPrs:    result.ReassignedPRs,
		SkippedPrs:       result.SkippedPRs,
	}, nil
}

func (h *Handler) GetTeamGet(ctx context.Context, request gen2.GetTeamGetRequestObject) (gen2.GetTeamGetResponseObject, error) {
	team, err := h.teamUseCase.GetTeam(ctx, request.Params.TeamName)
	if err != nil {
		if domainErr, ok := err.(*entity2.DomainError); ok && domainErr.Code == entity2.ErrorCodeNotFound {
			return gen2.GetTeamGet404JSONResponse{
				Error: struct {
					Code    gen2.ErrorResponseErrorCode `json:"code"`
					Message string                      `json:"message"`
				}{
					Code:    gen2.NOTFOUND,
					Message: domainErr.Message,
				},
			}, nil
		}
		return nil, err
	}

	// Конвертируем в gen
	genTeam := gen2.Team{
		TeamName: team.TeamName,
		Members:  make([]gen2.TeamMember, 0, len(team.Members)),
	}

	for _, member := range team.Members {
		genTeam.Members = append(genTeam.Members, gen2.TeamMember{
			UserId:   member.UserID,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}

	return gen2.GetTeamGet200JSONResponse(genTeam), nil
}

func (h *Handler) PostUsersSetIsActive(ctx context.Context, request gen2.PostUsersSetIsActiveRequestObject) (gen2.PostUsersSetIsActiveResponseObject, error) {
	if request.Body == nil {
		return gen2.PostUsersSetIsActive404JSONResponse{
			Error: struct {
				Code    gen2.ErrorResponseErrorCode `json:"code"`
				Message string                      `json:"message"`
			}{
				Code:    gen2.NOTFOUND,
				Message: "request body is required",
			},
		}, nil
	}

	user, err := h.userUseCase.SetUserIsActive(ctx, request.Body.UserId, request.Body.IsActive)
	if err != nil {
		if domainErr, ok := err.(*entity2.DomainError); ok && domainErr.Code == entity2.ErrorCodeNotFound {
			return gen2.PostUsersSetIsActive404JSONResponse{
				Error: struct {
					Code    gen2.ErrorResponseErrorCode `json:"code"`
					Message string                      `json:"message"`
				}{
					Code:    gen2.NOTFOUND,
					Message: domainErr.Message,
				},
			}, nil
		}
		return nil, err
	}

	return gen2.PostUsersSetIsActive200JSONResponse{
		User: &gen2.User{
			UserId:   user.UserID,
			Username: user.Username,
			TeamName: user.TeamName,
			IsActive: user.IsActive,
		},
	}, nil
}

func (h *Handler) PostPullRequestCreate(ctx context.Context, request gen2.PostPullRequestCreateRequestObject) (gen2.PostPullRequestCreateResponseObject, error) {
	if request.Body == nil {
		return gen2.PostPullRequestCreate404JSONResponse{
			Error: struct {
				Code    gen2.ErrorResponseErrorCode `json:"code"`
				Message string                      `json:"message"`
			}{
				Code:    gen2.NOTFOUND,
				Message: "request body is required",
			},
		}, nil
	}

	pr, err := h.pullRequestUseCase.CreatePullRequest(ctx, request.Body.PullRequestId, request.Body.PullRequestName, request.Body.AuthorId)
	if err != nil {
		if domainErr, ok := err.(*entity2.DomainError); ok {
			switch domainErr.Code {
			case entity2.ErrorCodePRExists:
				return gen2.PostPullRequestCreate409JSONResponse{
					Error: struct {
						Code    gen2.ErrorResponseErrorCode `json:"code"`
						Message string                      `json:"message"`
					}{
						Code:    gen2.PREXISTS,
						Message: domainErr.Message,
					},
				}, nil
			case entity2.ErrorCodeNotFound:
				return gen2.PostPullRequestCreate404JSONResponse{
					Error: struct {
						Code    gen2.ErrorResponseErrorCode `json:"code"`
						Message string                      `json:"message"`
					}{
						Code:    gen2.NOTFOUND,
						Message: domainErr.Message,
					},
				}, nil
			}
		}
		return nil, err
	}

	return gen2.PostPullRequestCreate201JSONResponse{
		Pr: entityToGenPullRequest(pr),
	}, nil
}

func (h *Handler) PostPullRequestMerge(ctx context.Context, request gen2.PostPullRequestMergeRequestObject) (gen2.PostPullRequestMergeResponseObject, error) {
	if request.Body == nil {
		return gen2.PostPullRequestMerge404JSONResponse{
			Error: struct {
				Code    gen2.ErrorResponseErrorCode `json:"code"`
				Message string                      `json:"message"`
			}{
				Code:    gen2.NOTFOUND,
				Message: "request body is required",
			},
		}, nil
	}

	pr, err := h.pullRequestUseCase.MergePullRequest(ctx, request.Body.PullRequestId)
	if err != nil {
		if domainErr, ok := err.(*entity2.DomainError); ok && domainErr.Code == entity2.ErrorCodeNotFound {
			return gen2.PostPullRequestMerge404JSONResponse{
				Error: struct {
					Code    gen2.ErrorResponseErrorCode `json:"code"`
					Message string                      `json:"message"`
				}{
					Code:    gen2.NOTFOUND,
					Message: domainErr.Message,
				},
			}, nil
		}
		return nil, err
	}

	return gen2.PostPullRequestMerge200JSONResponse{
		Pr: entityToGenPullRequest(pr),
	}, nil
}

func (h *Handler) PostPullRequestReassign(ctx context.Context, request gen2.PostPullRequestReassignRequestObject) (gen2.PostPullRequestReassignResponseObject, error) {
	if request.Body == nil {
		return gen2.PostPullRequestReassign404JSONResponse{
			Error: struct {
				Code    gen2.ErrorResponseErrorCode `json:"code"`
				Message string                      `json:"message"`
			}{
				Code:    gen2.NOTFOUND,
				Message: "request body is required",
			},
		}, nil
	}

	pr, replacedBy, err := h.pullRequestUseCase.ReassignReviewer(ctx, request.Body.PullRequestId, request.Body.OldUserId)
	if err != nil {
		if domainErr, ok := err.(*entity2.DomainError); ok {
			switch domainErr.Code {
			case entity2.ErrorCodeNotFound:
				return gen2.PostPullRequestReassign404JSONResponse{
					Error: struct {
						Code    gen2.ErrorResponseErrorCode `json:"code"`
						Message string                      `json:"message"`
					}{
						Code:    gen2.NOTFOUND,
						Message: domainErr.Message,
					},
				}, nil
			case entity2.ErrorCodePRMerged, entity2.ErrorCodeNotAssigned, entity2.ErrorCodeNoCandidate:
				return gen2.PostPullRequestReassign409JSONResponse{
					Error: struct {
						Code    gen2.ErrorResponseErrorCode `json:"code"`
						Message string                      `json:"message"`
					}{
						Code:    entityErrorCodeToGen(domainErr.Code),
						Message: domainErr.Message,
					},
				}, nil
			}
		}
		return nil, err
	}

	return gen2.PostPullRequestReassign200JSONResponse{
		Pr:         *entityToGenPullRequest(pr),
		ReplacedBy: replacedBy,
	}, nil
}

func (h *Handler) GetUsersGetReview(ctx context.Context, request gen2.GetUsersGetReviewRequestObject) (gen2.GetUsersGetReviewResponseObject, error) {
	prs, err := h.userUseCase.GetUserReviews(ctx, request.Params.UserId)
	if err != nil {
		if domainErr, ok := err.(*entity2.DomainError); ok && domainErr.Code == entity2.ErrorCodeNotFound {
			return gen2.GetUsersGetReview404JSONResponse{
				Error: struct {
					Code    gen2.ErrorResponseErrorCode `json:"code"`
					Message string                      `json:"message"`
				}{
					Code:    gen2.NOTFOUND,
					Message: domainErr.Message,
				},
			}, nil
		}
		return nil, err
	}

	genPRs := make([]gen2.PullRequestShort, 0, len(prs))
	for _, pr := range prs {
		genPRs = append(genPRs, gen2.PullRequestShort{
			PullRequestId:   pr.PullRequestID,
			PullRequestName: pr.PullRequestName,
			AuthorId:        pr.AuthorID,
			Status:          entityStatusToGenShort(pr.Status),
		})
	}

	return gen2.GetUsersGetReview200JSONResponse{
		UserId:       request.Params.UserId,
		PullRequests: genPRs,
	}, nil
}

func (h *Handler) GetStatsReviewers(ctx context.Context, _ gen2.GetStatsReviewersRequestObject) (gen2.GetStatsReviewersResponseObject, error) {
	stats, total, err := h.pullRequestUseCase.GetReviewerStats(ctx)
	if err != nil {
		return nil, err
	}

	response := gen2.ReviewerStatsResponse{
		Stats:        make([]gen2.ReviewerStat, 0, len(stats)),
		TotalReviews: total,
	}

	for _, stat := range stats {
		response.Stats = append(response.Stats, gen2.ReviewerStat{
			UserId:       stat.UserID,
			ReviewsCount: stat.ReviewsCount,
		})
	}

	return gen2.GetStatsReviewers200JSONResponse(response), nil
}

// Вспомогательные функции для конвертации

func entityToGenPullRequest(pr *entity2.PullRequest) *gen2.PullRequest {
	genPR := &gen2.PullRequest{
		PullRequestId:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorId:          pr.AuthorID,
		Status:            entityStatusToGen(pr.Status),
		AssignedReviewers: pr.AssignedReviewers,
		CreatedAt:         pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}
	return genPR
}

func entityStatusToGen(status entity2.PullRequestStatus) gen2.PullRequestStatus {
	switch status {
	case entity2.PullRequestStatusOpen:
		return gen2.PullRequestStatusOPEN
	case entity2.PullRequestStatusMerged:
		return gen2.PullRequestStatusMERGED
	default:
		return gen2.PullRequestStatusOPEN
	}
}

func entityStatusToGenShort(status entity2.PullRequestStatus) gen2.PullRequestShortStatus {
	switch status {
	case entity2.PullRequestStatusOpen:
		return gen2.PullRequestShortStatusOPEN
	case entity2.PullRequestStatusMerged:
		return gen2.PullRequestShortStatusMERGED
	default:
		return gen2.PullRequestShortStatusOPEN
	}
}

func entityErrorCodeToGen(code entity2.ErrorCode) gen2.ErrorResponseErrorCode {
	switch code {
	case entity2.ErrorCodeTeamExists:
		return gen2.TEAMEXISTS
	case entity2.ErrorCodePRExists:
		return gen2.PREXISTS
	case entity2.ErrorCodePRMerged:
		return gen2.PRMERGED
	case entity2.ErrorCodeNotAssigned:
		return gen2.NOTASSIGNED
	case entity2.ErrorCodeNoCandidate:
		return gen2.NOCANDIDATE
	case entity2.ErrorCodeNotFound:
		return gen2.NOTFOUND
	default:
		return gen2.NOTFOUND
	}
}
