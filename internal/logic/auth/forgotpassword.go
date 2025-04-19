package auth

import (
	"context"
	crand "crypto/rand" // Alias crypto/rand
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ForgotPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewForgotPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ForgotPasswordLogic {
	return &ForgotPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ForgotPasswordLogic) PostForgotPassword(c echo.Context, req *types.LoginCodeRequest) (resp *types.Response, err error) { // Assuming input type matches RequestLoginCodeRequest (just email)
	// 1. Find user by email
	// 1. Find user by email using model function
	userPtr, err := models.FindUserByEmail(l.svcCtx.DB, req.Email)
	if err != nil {
		// IMPORTANT: Do NOT reveal if the user exists or not for security reasons.
		// Always return a generic success message.
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			l.Errorf("Database error during forgot password for %s: %v", req.Email, err)
			// Still return generic success to user, but log the internal error
		} else {
			l.Infof("Forgot password request for non-existent email: %s", req.Email)
		}
		// Pretend success to the user
		return &types.Response{
			Success: true, // Indicate success even if user not found
			Message: "If an account with that email exists, password reset instructions have been sent.",
		}, nil
	}
	user := *userPtr // Dereference if found

	// 2. Generate secure random token
	tokenBytes := make([]byte, 32) // 32 bytes = 256 bits
	if _, err = crand.Read(tokenBytes); err != nil {
		l.Errorf("Failed to generate password reset token for %s: %v", req.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not process request (token gen)")
	}
	resetToken := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(1 * time.Hour) // Token valid for 1 hour

	// 3. Update user record with token and expiry
	// 3. Update user record using model function
	err = models.UpdateUserPasswordResetToken(l.svcCtx.DB, user.ID, resetToken, expiresAt)
	if err != nil {
		l.Errorf("Failed to save password reset token for user %s: %v", req.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not process request (db save)")
	}

	// 4. Send email (TODO: Implement email sending via svcCtx or dedicated service)
	frontendURL := l.svcCtx.Config.Email.BaseURL // Assuming BaseURL in config is the frontend URL
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"                                                                // Fallback for local dev - adjust if needed
		l.Info("FRONTEND_URL (svcCtx.Config.Email.BaseURL) not set, using default for password reset link.") // Use Info instead of Warn
	}
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, resetToken)
	// emailSubject := "Reset Your Password" // Commented out as unused for now
	// emailSubject := "Reset Your Password" // Keep commented out as unused for now
	emailBody := fmt.Sprintf("Click the link below to reset your password:\n\n%s\n\nIf you didn't request this, please ignore this email.", resetLink)

	// TODO: Replace this log with actual email sending call
	l.Infof("TODO: Send password reset email to %s. Link: %s", req.Email, resetLink)
	l.Infof("Email Body: %s", emailBody) // Log body for now
	// Example placeholder:
	// go func() {
	//  err := l.svcCtx.EmailService.Send(l.ctx, req.Email, emailSubject, emailBody)
	//  if err != nil {
	//      l.Errorf("Failed to send password reset email to %s: %v", req.Email, err)
	//  }
	// }()

	l.Infof("Password reset initiated for %s.", req.Email)

	// 5. Return generic success message
	// Optionally return token in dev environment if needed for testing, but avoid in prod.
	// For now, always return generic message.
	resp = &types.Response{
		Success: true,
		Message: "If an account with that email exists, password reset instructions have been sent.",
	}
	return resp, nil
}
