package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tarikstupac/chirpy/internal/auth"
	"github.com/tarikstupac/chirpy/internal/database"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

type UserCreateReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, req *http.Request) {

	userCreate := UserCreateReq{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&userCreate)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	if len(userCreate.Email) < 1 || !strings.Contains(userCreate.Email, "@") {
		respondWithError(w, http.StatusBadRequest, "Please enter a valid email", nil)
		return
	}

	if len(userCreate.Password) < 5 {
		respondWithError(w, http.StatusBadRequest, "Password must be at least 5 characters long", nil)
		return
	}
	hashedPassword, err := auth.HashPassword(userCreate.Password)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
	}

	createUserParams := database.CreateUserParams{Email: userCreate.Email, HashedPassword: hashedPassword}
	user, err := cfg.db.CreateUser(req.Context(), createUserParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating a user", err)
		return
	}
	newUser := User{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email, IsChirpyRed: user.IsChirpyRed}
	respondWithJSON(w, http.StatusCreated, newUser)
}

func (cfg *apiConfig) updateUserEmailPasswordHandler(w http.ResponseWriter, req *http.Request) {
	type userUpdateReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	userUpdate := userUpdateReq{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&userUpdate)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	if len(userUpdate.Email) < 1 || !strings.Contains(userUpdate.Email, "@") {
		respondWithError(w, http.StatusBadRequest, "Please enter a valid email", nil)
		return
	}

	if len(userUpdate.Password) < 5 {
		respondWithError(w, http.StatusBadRequest, "Password must be at least 5 characters long", nil)
		return
	}
	hashedPassword, err := auth.HashPassword(userUpdate.Password)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secretKey)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials", err)
		return
	}

	updateUserParams := database.UpdateUserEmailAndPasswordParams{Email: userUpdate.Email, HashedPassword: hashedPassword, ID: userID}
	user, err := cfg.db.UpdateUserEmailAndPassword(req.Context(), updateUserParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating a user", err)
		return
	}
	updatedUser := User{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email, IsChirpyRed: user.IsChirpyRed}
	respondWithJSON(w, http.StatusOK, updatedUser)
}
