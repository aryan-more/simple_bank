package api

import (
	"database/sql"
	"fmt"
	"net/http"

	db "github.com/aryan-more/simple_bank/db/sqlc"
	"github.com/gin-gonic/gin"
)

type transferRequest struct {
	FromAccountID int64  `json:"from_account" binding:"required,min=1"`
	ToAccountID   int64  `json:"to_account" binding:"required,min=1"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,oneof=USD EUR INR"`
}

func (server *Server) createTransfer(ctx *gin.Context) {
	var req transferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if !(server.validAccount(ctx, req.FromAccountID, req.Currency) && server.validAccount(ctx, req.ToAccountID, req.Currency)) {
		return
	}

	arg := db.TransferTxParams{
		Amount:        req.Amount,
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
	}

	result, err := server.store.TransferTX(ctx, arg)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, result)
}

func (server *Server) validAccount(ctx *gin.Context, accounID int64, currency string) bool {
	account, err := server.store.GetAccount(ctx, accounID)

	if err != nil {

		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return false
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return false

	}

	if account.Currency != currency {
		err := fmt.Errorf("account [%d] currency mismatch %s vs %s", account.ID, account.Currency, currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return false
	}

	return true
}
