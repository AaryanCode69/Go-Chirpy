package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/AaryanCode69/chirpy/internal/auth"
	"github.com/AaryanCode69/chirpy/internal/database"
	"github.com/google/uuid"
)

// User is what we send back to the client.
// It is separate from database.User so we control exactly what gets shown.
type User struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

// handlerCreateUser reads an email from the request and saves a new user.
func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}
	if params.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required", nil)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't Hash Password", err)
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create user", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, User{
		ID:          user.ID.String(),
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	})
}

func (cfg *apiConfig) handlerLoginUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	params := parameters{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}
	if params.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required", nil)
		return
	}

	user, err := cfg.db.FindUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password", err)
		return
	}

	isValid, err := auth.CheckPassword(params.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, 500, "Error checking passowrd", err)
		return
	}

	if !isValid {
		respondWithError(w, 401, "Incorrect email or password", err)
		return
	}

	jwtToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Duration(3600)*time.Second)
	if err != nil {
		respondWithError(w, 500, "Error Creating JWT", err)
		return
	}

	token := auth.MakeRefreshToken()
	refreshToken, err := cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(60 * 24 * time.Hour),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create RefreshToken", err)
		return
	}

	respondWithJSON(w, 200, User{
		ID:           user.ID.String(),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        jwtToken,
		RefreshToken: refreshToken.Token,
		IsChirpyRed:  user.IsChirpyRed,
	})
}

func (cfg *apiConfig) handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Invalid Refresh Token", err)
		return
	}

	refreshToken, err := cfg.db.GetRefreshTokenByToken(r.Context(), bearerToken)
	if err != nil {
		respondWithError(w, 401, "Invalid Refresh Token", err)
		return
	}

	if refreshToken.ExpiresAt.Before(time.Now().UTC()) || refreshToken.RevokedAt.Valid {
		respondWithError(w, 401, "Invalid Refresh Token", err)
		return
	}

	accessToken, err := auth.MakeJWT(refreshToken.UserID, cfg.jwtSecret, time.Duration(60)*time.Minute)
	if err != nil {
		respondWithError(w, 500, "Error Creating Access Token", err)
		return
	}

	respondWithJSON(w, 200, struct {
		Token string `json:"token"`
	}{
		Token: accessToken,
	})
}

func (cfg *apiConfig) handlerRevokeToken(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Invalid Refresh Token", err)
		return
	}

	refreshToken, err := cfg.db.GetRefreshTokenByToken(r.Context(), bearerToken)
	if err != nil {
		respondWithError(w, 401, "Invalid Refresh Token", err)
		return
	}

	_, err = cfg.db.RevokeTokenByToken(r.Context(), refreshToken.Token)
	if err != nil {
		respondWithError(w, 500, "Error Updating Refresh Token", err)
		return
	}

	respondWithJSON(w, 204, nil)
}

func (cfg *apiConfig) handleUserUpdate(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Invalid Refresh Token", err)
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, 401, "Invalid JWT", err)
		return
	}

	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	params := parameters{}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}
	if params.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required", nil)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't Hash Password", err)
		return
	}

	user, err := cfg.db.UpdateUserById(r.Context(), database.UpdateUserByIdParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
		ID:             userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't Save in DB", err)
		return
	}

	respondWithJSON(w, 200, User{
		ID:        user.ID.String(),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	})
}

func (cfg *apiConfig) handleUserUpgrade(w http.ResponseWriter, r *http.Request) {
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, 401, "Invalid ApiKey", err)
		return
	}

	if apiKey != cfg.polkaKey {
		respondWithError(w, 401, "Invalid ApiKey", err)
		return
	}

	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	params := parameters{}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	if params.Event != "user.upgraded" {
		respondWithJSON(w, 204, nil)
		return
	}

	id, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't decode UUID", err)
		return
	}

	_, err = cfg.db.UpgradeUserById(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user doesn't exist", err)
		return
	}

	respondWithJSON(w, 204, nil)
}
