package profile

import (
	"context"
	"errors"
	"net/http"

	"stab/middleware" // Added for ContextUserKey
	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm" // Added for gorm.ErrRecordNotFound
)

type ChangeEmailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangeEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangeEmailLogic {
	return &ChangeEmailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangeEmailLogic) PostChangeEmail(c echo.Context, req *types.ChangeEmailRequest) (resp *types.Response, err error) { // Added req parameter
	// 1. Get user from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for email change")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	user, ok := userCtx.(*models.User)
	if !ok {
		l.Errorf("User in context is not of type *models.User for email change")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}

	// Note: Validation of req happens automatically via go-zero/echo handler generation if tags are correct in types.go

	// 2. Verify current password
	if !user.CheckPassword(req.Password) {
		l.Infof("Incorrect password provided for email change attempt by user %s", user.Email)
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Incorrect password provided")
	}

	// 3. Check if new email is the same as the old one
	if user.Email == req.NewEmail {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "New email address cannot be the same as the current one")
	}

	// 4. Check if new email is already in use by another user
	var existingUser models.User
	err = l.svcCtx.DB.Where("email = ?", req.NewEmail).First(&existingUser).Error
	if err == nil {
		// Email found, means it's already in use
		l.Infof("Email change conflict for user %s: new email %s already exists.", user.Email, req.NewEmail)
		return nil, echo.NewHTTPError(http.StatusConflict, "This email address is already in use")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Database error during lookup
		l.Errorf("DB error checking new email %s for user %s: %v", req.NewEmail, user.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to verify new email address")
	}
	// If ErrRecordNotFound, the email is available, proceed.

	// 5. Update user's email
	// TODO: Consider sending a verification email to the new address before finalizing the change
	updateResult := l.svcCtx.DB.Model(&models.User{}).Where("id = ?", user.ID).Update("email", req.NewEmail)
	if updateResult.Error != nil {
		l.Errorf("Error updating email for user %s to %s: %v", user.Email, req.NewEmail, updateResult.Error)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to update email address")
	}
	if updateResult.RowsAffected == 0 {
		l.Infof("No rows affected when updating email for user %s (ID: %s)", user.Email, user.ID) // Changed from Warnf
		return nil, echo.NewHTTPError(http.StatusNotFound, "User not found during update")
	}

	// 6. Return success
	l.Infof("Successfully changed email for user %s to %s", user.ID, req.NewEmail)
	resp = &types.Response{
		Success: true,
		Message: "Email address updated successfully",
	}
	return resp, nil
}
