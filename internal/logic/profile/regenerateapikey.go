package profile

import (
	"context"
	"net/http"

	"stab/middleware" // Added for ContextUserKey
	"stab/models"
	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

type RegenerateApiKeyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegenerateApiKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegenerateApiKeyLogic {
	return &RegenerateApiKeyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PostRegenerateApiKey handles regenerating the API key for the authenticated user.
// It now returns a custom response type containing the new key.
func (l *RegenerateApiKeyLogic) PostRegenerateApiKey(c echo.Context) (resp *types.ApiKeyResponse, err error) { // Changed response type
	// 1. Get user from context
	userCtx := c.Get(middleware.ContextUserKey)
	if userCtx == nil {
		l.Errorf("User not found in context for API key regeneration")
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}
	user, ok := userCtx.(*models.User)
	if !ok {
		l.Errorf("User in context is not of type *models.User for API key regeneration")
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user data")
	}

	// 2. Generate New Key
	newApiKey, err := models.GenerateAPIKey()
	if err != nil {
		l.Errorf("Error generating new API key for user %s: %v", user.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate API key")
	}

	// 3. Update User Record in DB
	// 3. Update User Record in DB using model function
	err = models.UpdateUserAPIKey(l.svcCtx.DB, user.ID, newApiKey)
	if err != nil {
		l.Errorf("Error updating API key for user %s: %v", user.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to update API key")
	}
	// Note: The model function doesn't easily return RowsAffected.
	// If needed, the model function could be modified to return it,
	// or we could re-fetch the user to confirm the key changed, but that adds overhead.
	// Assuming success if no error is returned.

	l.Infof("Successfully regenerated API key for user %s", user.Email)

	// 4. Return New Key in the specific response type
	resp = &types.ApiKeyResponse{ // Use the specific response type
		ApiKey: newApiKey,
	}
	return resp, nil
}

// Define ApiKeyResponse in types.go if it doesn't exist:
// type ApiKeyResponse struct {
// 	ApiKey string `json:"apiKey"`
// }
