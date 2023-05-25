package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	db "github.com/aryan-more/simple_bank/db/sqlc"
	"github.com/aryan-more/simple_bank/util"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type userResponse struct {
	Username          string    `json:"username"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

func createUserResponse(user db.User) userResponse {
	return userResponse{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
	}
}

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

func (server *Server) createUser(ctx *gin.Context) {
	req := bindJson[createUserRequest](ctx)

	if req == nil {
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}

	user, err := server.store.CreateUser(ctx, arg)

	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			fmt.Println(pqErr.Code.Name())
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return

			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := createUserResponse(user)

	ctx.JSON(
		http.StatusOK,
		rsp,
	)

}

type logInUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=8"`
}

type logInUserResponse struct {
	AccessToken string       `json:"access_token"`
	User        userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {

	req := bindJson[logInUserRequest](ctx)
	if req == nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("Bad")))
		return
	}

	user, err := server.store.GetUser(ctx, req.Username)

	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = util.CheckedPassword(req.Password, user.HashedPassword)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, err := server.tokenMaker.CreateToken(req.Username, server.config.AccessTime)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := logInUserResponse{
		AccessToken: accessToken,
		User:        createUserResponse(user),
	}

	ctx.JSON(http.StatusOK, rsp)
}

func bindJson[T any](ctx *gin.Context) *T {
	body := new(T)
	if err := ctx.ShouldBindJSON(body); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return nil
	}
	return body
}
