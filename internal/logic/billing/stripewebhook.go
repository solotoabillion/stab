package billing

import (
	"context"
	"encoding/json"
	"errors" // Added for error checking
	"fmt"
	"io" // Added for reading body
	"net/http"
	"time"

	"stab/models"
	"stab/svc"
	"stab/types" // Added back for return type

	"github.com/google/uuid"
	"github.com/labstack/echo/v4" // Keep for HTTPError constants/creation
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	stripe "github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/subscription"
	webhook "github.com/stripe/stripe-go/v76/webhook"
)

type StripeWebhookLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStripeWebhookLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StripeWebhookLogic {
	return &StripeWebhookLogic{
		Logger: logx.WithContext(ctx), // Assign logger from context
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PostStripeWebhook processes incoming Stripe webhook events.
// It now reads the body and signature from the echo.Context.
func (l *StripeWebhookLogic) PostStripeWebhook(c echo.Context) (*types.Response, error) { // Reverted signature
	// 0. Read body and signature from context
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		l.Errorf("Failed to read Stripe webhook request body: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to read request body")
	}
	defer c.Request().Body.Close()

	signature := c.Request().Header.Get("Stripe-Signature")
	if signature == "" {
		l.Info("Missing Stripe-Signature header in webhook request")
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Missing Stripe-Signature header")
	}

	// 1. Get webhook secret from config
	endpointSecret := l.svcCtx.Config.Stripe.WebhookSecret // TODO: Verify this path in config struct
	if endpointSecret == "" {
		l.Error("STRIPE_WEBHOOK_SECRET missing from service config")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Webhook configuration error")
	}

	// 2. Verify the webhook signature
	event, err := webhook.ConstructEvent(payload, signature, endpointSecret)
	if err != nil {
		l.Infof("Webhook signature verification failed: %v", err)
		return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Webhook signature verification failed: %v", err))
	}

	l.Infof("Received Stripe webhook event: Type=%s, ID=%s", event.Type, event.ID)

	// 3. Initialize Stripe client (needed for fetching subscription details in some handlers)
	// TODO: Consider initializing Stripe client once in svcCtx if used frequently
	stripe.Key = l.svcCtx.Config.Stripe.SecretKey // TODO: Verify this path in config struct
	if stripe.Key == "" {
		l.Error("STRIPE_SECRET_KEY missing from service config for webhook processing")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Webhook configuration error")
	}

	// 4. Handle the event
	var processingError error
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			l.Errorf("Error parsing webhook JSON for checkout.session.completed: %v", err)
			processingError = echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook payload")
		} else {
			processingError = l.handleCheckoutSessionCompleted(session)
		}

	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			l.Errorf("Error parsing webhook JSON for customer.subscription.updated: %v", err)
			processingError = echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook payload")
		} else {
			processingError = l.handleSubscriptionUpdated(sub)
		}

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			l.Errorf("Error parsing webhook JSON for customer.subscription.deleted: %v", err)
			processingError = echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook payload")
		} else {
			processingError = l.handleSubscriptionDeleted(sub)
		}

	case "invoice.paid":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			l.Errorf("Error parsing webhook JSON for invoice.paid: %v", err)
			processingError = echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook payload")
		} else {
			processingError = l.handleInvoicePaid(invoice)
		}

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			l.Errorf("Error parsing webhook JSON for invoice.payment_failed: %v", err)
			processingError = echo.NewHTTPError(http.StatusBadRequest, "Error parsing webhook payload")
		} else {
			processingError = l.handleInvoicePaymentFailed(invoice)
		}

	default:
		l.Infof("Unhandled Stripe event type: %s", event.Type)
		// No error for unhandled types, just log and acknowledge receipt
	}

	// Return standard success response if no processing error occurred
	if processingError != nil {
		// The specific handlers already return echo.HTTPError or similar
		return nil, processingError
	}

	// Return success
	return &types.Response{Success: true, Message: "Webhook received successfully"}, nil
}

// --- Event Specific Handlers ---

func (l *StripeWebhookLogic) handleCheckoutSessionCompleted(session stripe.CheckoutSession) error {
	l.Infof("Processing checkout.session.completed: UserRef=%s, CustomerID=%s, SubscriptionID=%s", session.ClientReferenceID, session.Customer.ID, session.Subscription.ID)

	userIDStr := session.ClientReferenceID
	stripeCustomerID := ""
	if session.Customer != nil {
		stripeCustomerID = session.Customer.ID
	}
	stripeSubscriptionID := ""
	if session.Subscription != nil {
		stripeSubscriptionID = session.Subscription.ID
	}

	if userIDStr == "" || stripeCustomerID == "" || stripeSubscriptionID == "" {
		l.Errorf("Missing critical IDs in checkout.session.completed event: UserRef=%s, CustomerID=%s, SubscriptionID=%s", userIDStr, stripeCustomerID, stripeSubscriptionID)
		return echo.NewHTTPError(http.StatusBadRequest, "Webhook payload missing required IDs")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		l.Errorf("Invalid UserID UUID format in ClientReferenceID: %s, Error: %v", userIDStr, err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user identifier in webhook")
	}

	// Fetch Stripe Subscription details to get the Price ID
	sub, err := subscription.Get(stripeSubscriptionID, nil)
	if err != nil {
		l.Errorf("Failed to retrieve Stripe subscription %s: %v", stripeSubscriptionID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve subscription details from Stripe")
	}

	if len(sub.Items.Data) == 0 || sub.Items.Data[0].Price == nil {
		l.Errorf("Stripe subscription %s has no items or price information", stripeSubscriptionID)
		return echo.NewHTTPError(http.StatusBadRequest, "Subscription details missing plan information")
	}
	stripePriceID := sub.Items.Data[0].Price.ID

	// Find the internal plan corresponding to the Stripe Price ID
	var plan models.Plan
	var errPlan error
	// Use svcCtx.DB
	errPlan = l.svcCtx.DB.Where("stripe_price_id = ?", stripePriceID).Or("stripe_price_id_yearly = ?", stripePriceID).First(&plan).Error

	if errPlan != nil {
		if errors.Is(errPlan, gorm.ErrRecordNotFound) {
			l.Errorf("Failed to find Plan with Stripe Price ID %s in either monthly or yearly field", stripePriceID)
		} else {
			l.Errorf("Database error finding Plan with Stripe Price ID %s: %v", stripePriceID, errPlan)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not match subscription to an internal plan")
	}

	// Start DB transaction
	txErr := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		// Verify user exists and update Stripe Customer ID
		var userExists models.User
		// Use tx for transaction operations
		if err := tx.Select("id").First(&userExists, "id = ?", userID).Error; err != nil {
			l.Errorf("User %s not found during webhook processing: %v", userID, err)
			return echo.NewHTTPError(http.StatusBadRequest, "User specified in webhook not found")
		}
		if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("stripe_customer_id", stripeCustomerID).Error; err != nil {
			l.Errorf("Failed to update user %s with Stripe Customer ID %s: %v", userID, stripeCustomerID, err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update user billing info")
		}

		// Create or update the subscription record
		subscriptionRecord := models.Subscription{
			UserID:               userID,
			PlanID:               plan.ID,
			StripeSubscriptionID: stripeSubscriptionID,
			Status:               string(sub.Status),
			CurrentPeriodStart:   time.Unix(sub.CurrentPeriodStart, 0),
			CurrentPeriodEnd:     time.Unix(sub.CurrentPeriodEnd, 0),
			CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		}

		if err := tx.Where(models.Subscription{StripeSubscriptionID: stripeSubscriptionID}).
			Assign(subscriptionRecord).
			FirstOrCreate(&subscriptionRecord).Error; err != nil {
			l.Errorf("Failed to create/update subscription record for Stripe ID %s: %v", stripeSubscriptionID, err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save subscription details")
		}
		return nil // Commit transaction
	}) // End Transaction

	if txErr != nil {
		l.Errorf("Transaction failed for checkout.session.completed User %s, Subscription %s: %v", userID, stripeSubscriptionID, txErr)
		return txErr // Return the error bubbled up
	}

	l.Infof("Successfully processed checkout.session.completed for User %s, Subscription %s", userID, stripeSubscriptionID)
	return nil
}

func (l *StripeWebhookLogic) handleSubscriptionUpdated(sub stripe.Subscription) error {
	l.Infof("Processing customer.subscription.updated: ID=%s, Status=%s", sub.ID, sub.Status)

	txErr := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var existingSub models.Subscription
		if err := tx.Where("stripe_subscription_id = ?", sub.ID).First(&existingSub).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				l.Infof("Received customer.subscription.updated event for unknown subscription ID %s", sub.ID) // Changed Warnf to Infof
				return nil                                                                                     // Don't error, just ignore
			}
			l.Errorf("Failed to find subscription record for Stripe ID %s: %v", sub.ID, err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Database error finding subscription")
		}

		planID := existingSub.PlanID // Default to existing plan
		if len(sub.Items.Data) > 0 && sub.Items.Data[0].Price != nil {
			stripePriceID := sub.Items.Data[0].Price.ID
			var plan models.Plan
			errPlan := tx.Where("stripe_price_id = ?", stripePriceID).Or("stripe_price_id_yearly = ?", stripePriceID).First(&plan).Error
			if errPlan == nil {
				planID = plan.ID // Update plan ID if found
			} else if !errors.Is(errPlan, gorm.ErrRecordNotFound) {
				l.Errorf("DB error finding plan for price %s during subscription update %s: %v", stripePriceID, sub.ID, errPlan)
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify updated plan")
			} else {
				l.Errorf("Plan not found for price %s during subscription update %s. Keeping existing plan ID %s.", stripePriceID, sub.ID, planID) // Changed Warnf to Errorf as this indicates potential data issue
			}
		}

		updateData := map[string]interface{}{
			"status":               string(sub.Status),
			"current_period_start": time.Unix(sub.CurrentPeriodStart, 0),
			"current_period_end":   time.Unix(sub.CurrentPeriodEnd, 0),
			"cancel_at_period_end": sub.CancelAtPeriodEnd,
			"plan_id":              planID,
		}

		if err := tx.Model(&existingSub).Updates(updateData).Error; err != nil {
			l.Errorf("Failed to update subscription record for Stripe ID %s: %v", sub.ID, err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update subscription details")
		}
		return nil // Commit
	}) // End Transaction

	if txErr != nil {
		l.Errorf("Transaction failed for customer.subscription.updated %s: %v", sub.ID, txErr)
		return txErr
	}

	l.Infof("Successfully processed customer.subscription.updated for Subscription %s", sub.ID)
	return nil
}

func (l *StripeWebhookLogic) handleSubscriptionDeleted(sub stripe.Subscription) error {
	l.Infof("Processing customer.subscription.deleted: ID=%s", sub.ID)

	txErr := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		updateResult := tx.Model(&models.Subscription{}).
			Where("stripe_subscription_id = ?", sub.ID).
			Update("status", string(sub.Status)) // Use status from event, likely 'canceled'

		if updateResult.Error != nil {
			l.Errorf("Failed to update subscription status for deleted Stripe ID %s: %v", sub.ID, updateResult.Error)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update subscription status")
		}
		if updateResult.RowsAffected == 0 {
			l.Infof("Received customer.subscription.deleted event for unknown or already updated subscription ID %s", sub.ID) // Changed Warnf to Infof
			// Don't error, just ignore
		}
		return nil // Commit
	}) // End Transaction

	if txErr != nil {
		l.Errorf("Transaction failed for customer.subscription.deleted %s: %v", sub.ID, txErr)
		return txErr
	}

	l.Infof("Successfully processed customer.subscription.deleted for Subscription %s", sub.ID)
	return nil
}

func (l *StripeWebhookLogic) handleInvoicePaid(invoice stripe.Invoice) error {
	l.Infof("Processing invoice.paid: ID=%s, SubscriptionID=%s, Status=%s", invoice.ID, invoice.Subscription.ID, invoice.Status)

	if invoice.Status != stripe.InvoiceStatusPaid || invoice.Subscription == nil || invoice.Subscription.ID == "" {
		l.Infof("Ignoring invoice.paid event: ID=%s, Status=%s, SubscriptionID=%s (not paid or not linked to a subscription)", invoice.ID, invoice.Status, invoice.Subscription.ID)
		return nil // Ignore
	}
	stripeSubscriptionID := invoice.Subscription.ID

	txErr := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		updateResult := tx.Model(&models.Subscription{}).
			Where("stripe_subscription_id = ?", stripeSubscriptionID).
			Update("status", string(stripe.SubscriptionStatusActive))

		if updateResult.Error != nil {
			l.Errorf("Failed to update subscription status for invoice.paid event, Stripe Sub ID %s: %v", stripeSubscriptionID, updateResult.Error)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update subscription status")
		}
		if updateResult.RowsAffected == 0 {
			l.Infof("Received invoice.paid event for unknown subscription ID %s", stripeSubscriptionID) // Changed Warnf to Infof
			// Don't error, just ignore
		}
		return nil // Commit
	}) // End Transaction

	if txErr != nil {
		l.Errorf("Transaction failed for invoice.paid %s: %v", stripeSubscriptionID, txErr)
		return txErr
	}

	l.Infof("Successfully processed invoice.paid for Subscription %s", stripeSubscriptionID)
	return nil
}

func (l *StripeWebhookLogic) handleInvoicePaymentFailed(invoice stripe.Invoice) error {
	l.Infof("Processing invoice.payment_failed: ID=%s, SubscriptionID=%s, Status=%s", invoice.ID, invoice.Subscription.ID, invoice.Status) // Changed Warnf to Infof

	if invoice.Subscription == nil || invoice.Subscription.ID == "" {
		l.Infof("Ignoring invoice.payment_failed event: ID=%s, Status=%s (not linked to a subscription)", invoice.ID, invoice.Status)
		return nil // Ignore
	}
	stripeSubscriptionID := invoice.Subscription.ID

	txErr := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		updateResult := tx.Model(&models.Subscription{}).
			Where("stripe_subscription_id = ?", stripeSubscriptionID).
			Update("status", string(stripe.SubscriptionStatusPastDue))

		if updateResult.Error != nil {
			l.Errorf("Failed to update subscription status for invoice.payment_failed event, Stripe Sub ID %s: %v", stripeSubscriptionID, updateResult.Error)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update subscription status")
		}
		if updateResult.RowsAffected == 0 {
			l.Infof("Received invoice.payment_failed event for unknown subscription ID %s", stripeSubscriptionID) // Changed Warnf to Infof
			// Don't error, just ignore
		}
		// TODO: Optionally trigger a notification (e.g., email) to the user about the payment failure.
		return nil // Commit
	}) // End Transaction

	if txErr != nil {
		l.Errorf("Transaction failed for invoice.payment_failed %s: %v", stripeSubscriptionID, txErr)
		return txErr
	}

	l.Infof("Successfully processed invoice.payment_failed for Subscription %s, status set to past_due", stripeSubscriptionID) // Changed Warnf to Infof
	return nil
}
