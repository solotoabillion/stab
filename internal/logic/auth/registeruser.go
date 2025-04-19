package auth

import (
	"context"
	"encoding/json" // Added for JSON handling
	"errors"
	"fmt"
	mrand "math/rand" // Alias math/rand
	"net/http"
	"regexp" // Added import
	"strings"
	"time"

	"github.com/solotoabillion/stab/models"
	"github.com/solotoabillion/stab/svc"
	"github.com/solotoabillion/stab/types"

	"github.com/labstack/echo/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// Removed package-level subdomain generation helpers

type RegisterUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterUserLogic {
	return &RegisterUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterUserLogic) PostRegisterUser(c echo.Context, req *types.RegisterRequest) (resp *types.Response, err error) { // Changed response type to types.Response as per API def
	// 1. Check if user already exists
	// 1. Check if user already exists using model function
	_, err = models.FindUserByEmail(l.svcCtx.DB, req.Email)
	if err == nil {
		// User found (no error means user exists)
		l.Errorf("User registration conflict for email: %s", req.Email)
		return nil, echo.NewHTTPError(http.StatusConflict, "User with this email already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Other DB error during lookup
		l.Errorf("Database error checking user existence for %s: %v", req.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Database error checking user")
	}
	// User does not exist (ErrRecordNotFound), proceed.

	// 2. Prepare ProfileData
	profileData := types.UserProfileData{
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		AuthProvider: "local", // Standard registration is 'local' provider
	}
	profileJSON, err := json.Marshal(profileData)
	if err != nil {
		l.Errorf("Error marshalling profile data for %s: %v", req.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not process registration (profile data)")
	}

	// 3. Create new user model and hash password
	user := models.User{
		Email:       req.Email,
		Role:        models.SystemRoleUser, // Set default role
		ProfileData: profileJSON,           // Assign marshalled JSON
	}
	if err := user.SetPassword(req.Password); err != nil {
		l.Errorf("Error hashing password for %s: %v", req.Email, err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not process registration (password hash)")
	}

	// 4. Generate API Key and Default Subdomain (using hooks in model now)
	// ApiKey and DefaultSubdomain are handled by BeforeCreate hook if needed
	// user.ApiKey = apiKey // Handled by hook
	// user.DefaultSubdomain = generateRandomSubdomain() // Handled by hook? Or generate here? Let's generate here for now.
	// TODO: Add uniqueness check loop for subdomain if needed
	// --- Subdomain Generation (Moved inside function) ---
	generateRandomSubdomain := func() string {
		// Word lists defined locally
		adjectives := []string{"tiny", "blue", "red", "fast", "slow", "calm", "warm", "cool", "dark", "soft", "loud", "new", "old", "good", "nice", "fine", "fair", "kind", "keen", "bold", "brave", "calm", "eager", "glad", "jolly", "merry", "proud", "silly", "witty", "zany"}
		nouns := []string{"cat", "dog", "fox", "bird", "fish", "frog", "bear", "wolf", "tree", "leaf", "rock", "star", "moon", "sun", "ship", "boat", "car", "bus", "bike", "road", "path", "hill", "lake", "pond", "desk", "lamp", "book", "pen", "cup", "dish"}
		nonAlphanumericRegex := regexp.MustCompile(`[^a-z0-9]+`)
		dashRegex := regexp.MustCompile(`-{2,}`)

		filterWords := func(words []string) []string {
			filtered := []string{}
			for _, w := range words {
				if len(w) >= 3 && len(w) <= 5 {
					filtered = append(filtered, w)
				}
			}
			if len(filtered) > 0 {
				mrand.New(mrand.NewSource(time.Now().UnixNano()))
			}
			return filtered
		}
		validAdjectives := filterWords(adjectives)
		validNouns := filterWords(nouns)

		if len(validAdjectives) < 2 || len(validNouns) == 0 {
			return fmt.Sprintf("user-%d", mrand.Intn(10000))
		}

		word1 := validAdjectives[mrand.Intn(len(validAdjectives))]
		word2 := validAdjectives[mrand.Intn(len(validAdjectives))]
		for word1 == word2 {
			word2 = validAdjectives[mrand.Intn(len(validAdjectives))]
		}
		word3 := validNouns[mrand.Intn(len(validNouns))]

		// Use strings package locally now
		lower := strings.ToLower(fmt.Sprintf("%s-%s-%s", word1, word2, word3))
		noSpecial := nonAlphanumericRegex.ReplaceAllString(lower, "-")
		noMultipleDashes := dashRegex.ReplaceAllString(noSpecial, "-")
		trimmed := strings.Trim(noMultipleDashes, "-")
		return trimmed
	}
	// Need to re-add strings import now
	user.DefaultSubdomain = generateRandomSubdomain()
	// --- End Subdomain Generation ---

	// 5. Save user to database
	// The BeforeCreate hook in models/user.go will generate UUID and API key
	// 5. Save user to database using model function
	if err := models.CreateUser(l.svcCtx.DB, &user); err != nil {
		// TODO: Handle potential unique constraint violation on DefaultSubdomain if generation collides
		l.Errorf("Error creating user %s: %v", req.Email, err)
		// Check for specific unique constraint errors if the DB driver supports it
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Could not register user (database create)")
	}

	l.Infof("User registered successfully: %s, Subdomain: %s", user.Email, user.DefaultSubdomain)

	// 6. Return success response (API defines types.Response, not types.User)
	resp = &types.Response{
		Success: true,
		Message: "User registered successfully",
	}

	return resp, nil // Return success response and nil error
}

// Removed local derefString helper function
