package billing

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"stab/middleware"
	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	stripe "github.com/stripe/stripe-go/v76"
	invoiceapi "github.com/stripe/stripe-go/v76/invoice"
	"github.com/zeromicro/go-zero/core/logx"
)

type InvoicesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewInvoicesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InvoicesLogic {
	return &InvoicesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *InvoicesLogic) GetInvoices(c echo.Context) (resp *types.InvoiceResponse, err error) {
	// 1. Get authenticated User from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for getting invoices")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	authedUser, ok := userCtx.(*models.User)
	if !ok || authedUser == nil {
		l.Errorf("User in context is not of type *models.User or is nil")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}
	userID := authedUser.ID

	// 2. Check if user has a Stripe Customer ID
	if authedUser.StripeCustomerID == nil || *authedUser.StripeCustomerID == "" {
		l.Infof("User %s attempted to fetch invoices without a Stripe Customer ID", userID.String())
		// Return success with empty list as per old handler logic
		return &types.InvoiceResponse{
			Success:  true,
			Message:  "No billing account found.",
			Invoices: []types.InvoiceItem{},
		}, nil
	}
	stripeCustomerID := *authedUser.StripeCustomerID

	// 3. Initialize Stripe client
	stripe.Key = l.svcCtx.Config.Stripe.SecretKey
	if stripe.Key == "" {
		l.Error("STRIPE_SECRET_KEY missing from service config for GetInvoices")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Billing configuration error")
	}

	// 4. Prepare Stripe List Params (Pagination from query params)
	params := &stripe.InvoiceListParams{
		Customer: stripe.String(stripeCustomerID),
		ListParams: stripe.ListParams{
			Limit: stripe.Int64(10), // Default limit
		},
	}

	limitStr := c.QueryParam("limit") // Assuming limit is passed as query param
	if limitStr != "" {
		limit, parseErr := strconv.ParseInt(limitStr, 10, 64)
		if parseErr == nil && limit > 0 && limit <= 100 { // Basic validation
			params.Limit = stripe.Int64(limit)
		} else {
			l.Infof("Invalid 'limit' query parameter received: %s", limitStr)
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid 'limit' parameter. Must be between 1 and 100.")
		}
	}

	startingAfter := c.QueryParam("starting_after") // Assuming starting_after is passed as query param
	if startingAfter != "" {
		params.StartingAfter = stripe.String(startingAfter)
	}

	// 5. List Invoices from Stripe
	i := invoiceapi.List(params)

	// 6. Iterate and Map to types.InvoiceItem
	invoiceList := make([]types.InvoiceItem, 0)
	for i.Next() {
		inv := i.Invoice()
		// Skip draft invoices? Or include them? Old code didn't filter. Including all for now.
		invoiceList = append(invoiceList, types.InvoiceItem{
			ID:        inv.ID,
			Amount:    float64(inv.AmountPaid) / 100.0, // Assuming AmountPaid is what's relevant
			Currency:  string(inv.Currency),
			Status:    string(inv.Status),
			CreatedAt: time.Unix(inv.Created, 0).Format(time.RFC3339), // Format timestamp
			// types.InvoiceItem doesn't include PDF/Hosted URLs
		})
	}

	// 8. Check for iterator errors
	if err := i.Err(); err != nil {
		l.Errorf("Error listing Stripe invoices for customer %s: %v", stripeCustomerID, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve invoice history: %s", err.Error()))
	}

	// Ensure we return an empty slice, not nil
	if invoiceList == nil {
		invoiceList = []types.InvoiceItem{}
	}

	// 9. Populate and return response
	resp = &types.InvoiceResponse{
		Success:  true,
		Message:  "Invoices retrieved successfully",
		Invoices: invoiceList,
	}

	l.Infof("Retrieved %d invoices for user %s (customer %s)", len(invoiceList), userID.String(), stripeCustomerID)
	return resp, nil
}
