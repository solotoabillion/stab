package admin

import (
	"context"
	"net/http" // Added

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetDashboardMetricsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDashboardMetricsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDashboardMetricsLogic {
	return &GetDashboardMetricsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDashboardMetricsLogic) GetDashboardMetrics(c echo.Context) (resp *types.DashboardMetrics, err error) {
	l.Logger.Info("Admin request: GetDashboardMetrics")

	var userCount int64
	var activeSubCount int64
	var teamCount int64
	// var blogPostCount int64 // Not in the response type

	// Count Users (non-admin)
	userCount, err = models.CountNonAdminUsers(l.svcCtx.DB)
	if err != nil {
		l.Logger.Errorf("Admin: Error counting users: %v", err)
		// Decide if we should return partial data or fail the request
		// For now, let's fail if any count fails
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to count users")
	}

	// Count Active Subscriptions
	activeSubCount, err = models.CountActiveSubscriptions(l.svcCtx.DB)
	if err != nil {
		l.Logger.Errorf("Admin: Error counting active subscriptions: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to count subscriptions")
	}

	// Count Teams
	teamCount, err = models.CountTeams(l.svcCtx.DB)
	if err != nil {
		l.Logger.Errorf("Admin: Error counting teams: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to count teams")
	}

	// Count Blog Posts (if needed later)
	// blogPostCount, err = models.CountBlogPosts(l.svcCtx.DB)
	// if err != nil {
	// 	l.Logger.Errorf("Admin: Error counting blog posts: %v", err)
	// 	return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to count blog posts")
	// }

	// Prepare response based on types.DashboardMetrics
	resp = &types.DashboardMetrics{
		TotalUsers:          int(userCount),      // Convert int64 to int
		ActiveUsers:         0,                   // TODO: Need logic/criteria for 'active' users
		TotalTeams:          int(teamCount),      // Convert int64 to int
		ActiveSubscriptions: int(activeSubCount), // Convert int64 to int
	}

	l.Logger.Infof("Admin: Retrieved dashboard metrics - Users: %d, Active Subs: %d, Teams: %d", resp.TotalUsers, resp.ActiveSubscriptions, resp.TotalTeams)
	return resp, nil
}
