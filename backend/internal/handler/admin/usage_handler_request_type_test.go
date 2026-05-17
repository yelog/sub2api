package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminUsageRepoCapture struct {
	service.UsageLogRepository
	listParams       pagination.PaginationParams
	listFilters      usagestats.UsageLogFilters
	statsFilters     usagestats.UsageLogFilters
	allStatsFilters  usagestats.UsageLogFilters
	fastStatsFilters usagestats.UsageLogFilters
	statsCalls       int
}

func (s *adminUsageRepoCapture) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	s.listParams = params
	s.listFilters = filters
	return []service.UsageLog{}, &pagination.PaginationResult{
		Total:    0,
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    0,
	}, nil
}

func (s *adminUsageRepoCapture) GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	s.statsCalls++
	s.statsFilters = filters
	if filters.OpenAIFastBucket == usagestats.OpenAIFastBucketAll {
		s.allStatsFilters = filters
	}
	if filters.OpenAIFastBucket == usagestats.OpenAIFastBucketFast {
		s.fastStatsFilters = filters
	}
	return &usagestats.UsageStats{TotalRequests: int64(s.statsCalls)}, nil
}

func newAdminUsageRequestTypeTestRouter(repo *adminUsageRepoCapture) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(repo, nil, nil, nil)
	handler := NewUsageHandler(usageSvc, nil, nil, nil)
	router := gin.New()
	router.GET("/admin/usage", handler.List)
	router.GET("/admin/usage/stats", handler.Stats)
	return router
}

func TestAdminUsageListRequestTypePriority(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage?request_type=ws_v2&stream=false", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, repo.listFilters.RequestType)
	require.Equal(t, int16(service.RequestTypeWSV2), *repo.listFilters.RequestType)
	require.Nil(t, repo.listFilters.Stream)
}

func TestAdminUsageListInvalidRequestType(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage?request_type=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminUsageListInvalidStream(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage?stream=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminUsageListExactTotalTrue(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage?exact_total=true", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, repo.listFilters.ExactTotal)
}

func TestAdminUsageListInvalidExactTotal(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage?exact_total=oops", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminUsageStatsRequestTypePriority(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/stats?request_type=stream&stream=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, repo.statsFilters.RequestType)
	require.Equal(t, int16(service.RequestTypeStream), *repo.statsFilters.RequestType)
	require.Nil(t, repo.statsFilters.Stream)
}

func TestAdminUsageStatsOpenAIFastBucketFilter(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/stats?openai_fast=fast", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, usagestats.OpenAIFastBucketFast, repo.statsFilters.OpenAIFastBucket)
}

func TestAdminUsageStatsOpenAINonFastBucketFilter(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/stats?openai_fast=non_fast", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, usagestats.OpenAIFastBucketNonFast, repo.statsFilters.OpenAIFastBucket)
}

func TestAdminUsageStatsInvalidOpenAIFastBucket(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/stats?openai_fast=priority", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminUsageStatsDefaultIncludesFastAndNonFastBreakdown(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 3, repo.statsCalls)
	require.Equal(t, usagestats.OpenAIFastBucketAll, repo.allStatsFilters.OpenAIFastBucket)
	require.Equal(t, usagestats.OpenAIFastBucketFast, repo.fastStatsFilters.OpenAIFastBucket)
	require.Contains(t, rec.Body.String(), "fast_stats")
	require.Contains(t, rec.Body.String(), "non_fast_stats")
}

func TestAdminUsageStatsBucketFilterDoesNotFetchBreakdown(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/stats?openai_fast=fast", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, repo.statsCalls)
	require.NotContains(t, rec.Body.String(), "fast_stats")
	require.NotContains(t, rec.Body.String(), "non_fast_stats")
}

func TestAdminUsageStatsInvalidRequestType(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/stats?request_type=oops", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAdminUsageStatsInvalidStream(t *testing.T) {
	repo := &adminUsageRepoCapture{}
	router := newAdminUsageRequestTypeTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/stats?stream=oops", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
