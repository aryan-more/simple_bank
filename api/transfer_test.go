package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/aryan-more/simple_bank/db/mock"
	db "github.com/aryan-more/simple_bank/db/sqlc"
	"github.com/aryan-more/simple_bank/token"
	"github.com/aryan-more/simple_bank/util"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestTransferAPI(t *testing.T) {
	username1 := util.RandomString(5)
	username2 := util.RandomString(5)

	currency := util.RandomCurrency()

	user1 := randomAccountWithCurrency(username1, currency)
	user2 := randomAccountWithCurrency(username2, currency)
	amount := 10

	invalidCurrency := currency
	for currency == invalidCurrency {
		invalidCurrency = util.RandomCurrency()
	}
	user3 := randomAccountWithCurrency(util.RandomOwner(), invalidCurrency)

	testcase := []struct {
		name          string
		body          gin.H
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStub     func(store *mockdb.MockStore)
		responseCheck func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "Ok",
			body: gin.H{
				"from_account": user1.ID,
				"to_account":   user2.ID,
				"amount":       amount,
				"currency":     currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user1.ID)).Times(1).Return(user1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user2.ID)).Times(1).Return(user2, nil)

				arg := db.TransferTxParams{
					Amount:        int64(amount),
					FromAccountID: user1.ID,
					ToAccountID:   user2.ID,
				}

				store.EXPECT().TransferTX(gomock.Any(), gomock.Eq(arg)).Times(1)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, username1, time.Minute)
			},
		},
		{
			name: "TransactionFailed",
			body: gin.H{
				"from_account": user1.ID,
				"to_account":   user2.ID,
				"amount":       amount,
				"currency":     currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user1.ID)).Times(1).Return(user1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user2.ID)).Times(1).Return(user2, nil)

				arg := db.TransferTxParams{
					Amount:        int64(amount),
					FromAccountID: user1.ID,
					ToAccountID:   user2.ID,
				}

				store.EXPECT().TransferTX(gomock.Any(), gomock.Eq(arg)).Times(1).Return(db.TransferTxResult{}, sql.ErrConnDone)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, username1, time.Minute)
			},
		},
		{
			name: "Invalid body",
			body: gin.H{},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user1.ID)).Times(0)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user2.ID)).Times(0)

				store.EXPECT().TransferTX(gomock.Any(), gomock.Any()).Times(0)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, username1, time.Minute)
			},
		},
		{
			name: "UnauthorizedUser",
			body: gin.H{
				"from_account": user1.ID,
				"to_account":   user2.ID,
				"amount":       10,
				"currency":     currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user1.ID)).Times(1).Return(user1, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user2.ID)).Times(0)

				store.EXPECT().TransferTX(gomock.Any(), gomock.Any()).Times(0)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, username2, time.Minute)
			},
		},
		{
			name: "InvalidCurrencySender",
			body: gin.H{
				"from_account": user3.ID,
				"to_account":   user2.ID,
				"amount":       10,
				"currency":     currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user3.ID)).Times(1).Return(user3, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user2.ID)).Times(0)

				store.EXPECT().TransferTX(gomock.Any(), gomock.Any()).Times(0)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, user3.Owner, time.Minute)
			},
		},
		{
			name: "InvalidCurrencyReceiver",
			body: gin.H{
				"from_account": user1.ID,
				"to_account":   user3.ID,
				"amount":       10,
				"currency":     currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user3.ID)).Times(1).Return(user3, nil)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user1.ID)).Times(1).Return(user1, nil)

				store.EXPECT().TransferTX(gomock.Any(), gomock.Any()).Times(0)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, username1, time.Minute)
			},
		},
		{
			name: "UserNotFound",
			body: gin.H{
				"from_account": user1.ID,
				"to_account":   user2.ID,
				"amount":       amount,
				"currency":     currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user1.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user2.ID)).Times(0)

				store.EXPECT().TransferTX(gomock.Any(), gomock.Any()).Times(0)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, username1, time.Minute)
			},
		},
		{
			name: "UserNotFound",
			body: gin.H{
				"from_account": user1.ID,
				"to_account":   user2.ID,
				"amount":       amount,
				"currency":     currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user1.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(user2.ID)).Times(0)

				store.EXPECT().TransferTX(gomock.Any(), gomock.Any()).Times(0)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, username1, time.Minute)
			},
		},
	}

	for _, tc := range testcase {

		t.Run(
			tc.name,
			func(t *testing.T) {

				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				store := mockdb.NewMockStore(ctrl)
				tc.buildStub(store)
				server := newTestServer(t, store)

				data, err := json.Marshal(tc.body)
				require.NoError(t, err)

				recorder := httptest.NewRecorder()

				url := "/transfers"
				req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
				require.NoError(t, err)
				tc.setupAuth(t, req, server.tokenMaker)

				server.router.ServeHTTP(recorder, req)
				tc.responseCheck(t, recorder)

			},
		)

	}

}
