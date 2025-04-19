package auth

import (
	"context"
	"errors"
	"net/http"

	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ResetPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetPasswordLogic {
	return &ResetPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetPasswordLogic) PostResetPassword(c echo.Context, req *types.ResetPasswordRequest) (resp *types.Response, err error) { // Assuming input type matches types.ResetPasswordRequest
	// 1. Find user by the non-null, non-expired token
	// 1. Find user by valid reset token using model function
	userPtr, err := models.FindUserByValidPasswordResetToken(l.svcCtx.DB, req.Token)
	if err != nil {
		// Handles token not found, expired token, or other DB errors
		if errors.Is(err, gorm.ErrRecordNotFound) {
			l.Info("Invalid or expired password reset token presented.")
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid or expired password reset token")
		}
		l.Errorf("Database error during password reset token lookup: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not process request (db lookup)")
	}
	user := *userPtr // Dereference if found

	// 2. Token is valid, update the password
	if err := user.SetPassword(req.Password); err != nil {
		l.Errorf("Error hashing new password during reset for user %s: %v", user.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not update password")
	}

	// 3. Clear the reset token fields
	// 3. Update password and clear token fields using model function
	if err := models.UpdateUserPasswordAndClearToken(l.svcCtx.DB, user.ID, user.Password); err != nil {
		l.Errorf("Failed to update user password and clear token for user %s: %v", user.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not finalize password reset")
	}

	l.Infof("Password successfully reset for user %s", user.Email)

	// 5. Return success response
	resp = &types.Response{
		Success: true,
		Message: "Password has been successfully reset.",
	}
	return resp, nil
}
