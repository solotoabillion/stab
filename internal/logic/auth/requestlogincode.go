package auth

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"stab/svc"
	"stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	loginCodeExpiry = 5 * time.Minute // Login codes are valid for 5 minutes
)

type RequestLoginCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRequestLoginCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RequestLoginCodeLogic {
	// No need to seed the global rand generator in Go 1.20+
	return &RequestLoginCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// generateLoginCode creates a random 6-digit numeric code
func generateLoginCode() string {
	// Use the default global rand source
	code := rand.Intn(1000000)       // Generate a number between 0 and 999999
	return fmt.Sprintf("%06d", code) // Pad with leading zeros if needed
}

func (l *RequestLoginCodeLogic) PostRequestLoginCode(c echo.Context, req *types.LoginCodeRequest) (resp *types.Response, err error) {
	l.Logger.Infof("RequestLoginCode attempt for email: %s", req.Email)

	// 1. Generate secure code
	loginCode := generateLoginCode()

	// 2. Store code (Skipped - No Redis caching in this migration phase)
	//    The verification step will need adjustment later.
	l.Logger.Infof("Login code generated for %s: %s (Not stored persistently in this phase)", req.Email, loginCode)

	// 3. Send code via email (Placeholder/Log for now)
	emailSubject := "Your Login Code"
	emailBody := fmt.Sprintf("Your login code is: %s\n\nIt will expire in %d minutes.", loginCode, int(loginCodeExpiry.Minutes()))

	// TODO: Implement actual email sending using a communication service/library
	// Example: err = l.svcCtx.EmailService.Send(req.Email, emailSubject, emailBody)
	// if err != nil { l.Logger.Errorf(...) /* Handle error, maybe still return success? */ }
	l.Logger.Infof("[Action Required] Send email to %s - Subject: %s - Body: %s", req.Email, emailSubject, emailBody)
	// Use Errorf for logging warnings/errors if Warnf is not available
	l.Logger.Errorf("Login code for %s: %s (Logging for testing/dev, not stored)", req.Email, loginCode)

	// 4. Return generic success response
	resp = &types.Response{
		Success: true,
		Message: "If an account with that email exists, a login code has been sent.",
	}

	l.Logger.Infof("Login code generated and email sending simulated for %s.", req.Email)
	return resp, nil
}
