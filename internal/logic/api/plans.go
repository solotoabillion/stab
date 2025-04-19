package api

import (
	"context"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type PlansLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPlansLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PlansLogic {
	return &PlansLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PlansLogic) GetPlans(c echo.Context) (resp *[]types.Plan, err error) {
	resp = &[]types.Plan{}
	var plans []models.Plan
	plans, err = models.FindActivePlans(l.svcCtx.DB)
	if err != nil {
		return nil, err
	}
	for _, plan := range plans {
		*resp = append(*resp, types.Plan{
			ID:           plan.ID,
			Name:         plan.Name,
			PriceMonthly: plan.PriceMonthly,
			PriceYearly: func() float64 {
				if plan.PriceYearly != nil {
					return *plan.PriceYearly
				}
				return 0
			}(),
			Features: models.UnmarshalJSONFeatures(plan.Features),
		})
	}
	return resp, nil
}
