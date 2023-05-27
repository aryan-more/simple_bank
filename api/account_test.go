package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestListAccountAPI(t *testing.T) {

	n := 5
	accounts := make([]db.Account, n)
	owner := util.RandomString(5)
	for index := 0; index < n; index++ {
		accounts[index] = randomAccount(owner)
	}

	type Query struct {
		pageID   int
		pageSize int
	}

	testcase := []struct {
		name          string
		query         Query
		buildStub     func(store *mockdb.MockStore)
		responseCheck func(t *testing.T, recorder *httptest.ResponseRecorder)
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
	}{
		{
			name: "Ok",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			buildStub: func(store *mockdb.MockStore) {
				arg := db.ListAccountsParams{
					Limit:  int32(n),
					Offset: 0,
					Owner:  owner,
				}

				store.EXPECT().ListAccounts(gomock.Any(), gomock.Eq(arg)).Times(1).Return(accounts, nil)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireMatchAccounts(t, recorder.Body, accounts)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)
			},
		},

		{
			name: "NoTokenHeader",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			buildStub: func(store *mockdb.MockStore) {

				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(0)

			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
		},
		{
			name: "Invalid PageID",
			query: Query{
				pageID:   -1,
				pageSize: n,
			},
			buildStub: func(store *mockdb.MockStore) {

				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(0)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)

			},
		},
		{
			name: "Invalid PageSize",
			query: Query{
				pageID:   1,
				pageSize: 0,
			},
			buildStub: func(store *mockdb.MockStore) {

				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(0)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)

			},
		},
		{
			name: "Internal Server Error",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			buildStub: func(store *mockdb.MockStore) {

				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(1).Return(accounts, sql.ErrConnDone)
			},
			responseCheck: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)

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

				url := fmt.Sprintf("/accounts")
				req, err := http.NewRequest(http.MethodGet, url, nil)
				require.NoError(t, err)

				q := req.URL.Query()
				q.Add("page_id", fmt.Sprintf("%d", tc.query.pageID))
				q.Add("page_size", fmt.Sprintf("%d", tc.query.pageSize))
				req.URL.RawQuery = q.Encode()
				tc.setupAuth(t, req, server.tokenMaker)

				recorder := httptest.NewRecorder()
				server.router.ServeHTTP(recorder, req)
				tc.responseCheck(t, recorder)

			},
		)
	}

}

func TestCreateAccount(t *testing.T) {
	owner := util.RandomString(5)
	account := randomAccountZeroBalance(owner)
	testcase := []struct {
		name          string
		body          gin.H
		buildStub     func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
	}{
		{
			name: "OK",
			body: gin.H{
				"owner":    account.Owner,
				"currency": account.Currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireMatchAccount(t, recorder.Body, account)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)
			},
		},
		{
			name: "InvalidUserToken",
			body: gin.H{
				"owner":    account.Owner,
				"currency": account.Currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, util.RandomString(10), time.Minute)
			},
		},
		{
			name: "NoTokenHeader",
			body: gin.H{
				"owner":    account.Owner,
				"currency": account.Currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
		},
		{
			name: "Internal Server Error",
			body: gin.H{
				"owner":    account.Owner,
				"currency": account.Currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(1).Return(account, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)
			},
		},
		{
			name: "Invalid currency",
			body: gin.H{
				"owner":    account.Owner,
				"currency": "INR",
			},
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)
			},
		},
		{
			name: "Owner Doesn't exist",
			body: gin.H{
				"owner":    account.Owner,
				"currency": account.Currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.
					EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					Times(1).
					Return(
						account,
						&pq.Error{
							Code: pq.ErrorCode("23503"),
						},
					)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)
			},
		},
		{
			name: "Duplicate Account",
			body: gin.H{
				"owner":    account.Owner,
				"currency": account.Currency,
			},
			buildStub: func(store *mockdb.MockStore) {
				store.
					EXPECT().
					CreateAccount(gomock.Any(), gomock.Any()).
					Times(1).
					Return(
						account,
						&pq.Error{
							Code: pq.ErrorCode("23505"),
						},
					)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)
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
				url := "/accounts"
				req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
				tc.setupAuth(t, req, server.tokenMaker)

				require.NoError(t, err)
				server.router.ServeHTTP(recorder, req)

				tc.checkResponse(t, recorder)
			},
		)
	}

}

func TestGetAccount(t *testing.T) {
	owner := util.RandomString(5)
	account := randomAccount(owner)

	testcase := []struct {
		name          string
		accountID     int64
		buildStub     func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
	}{
		{
			name:      "Ok",
			accountID: account.ID,
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				account.CreatedAt = account.CreatedAt.Local()
				requireMatchAccount(t, recorder.Body, account)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)
			},
		},
		{
			name: "InvalidUserToken",

			accountID: account.ID,
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, util.RandomString(10), time.Minute)
			},
		},
		{
			name:      "NoTokenHeader",
			accountID: account.ID,
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(0)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)

			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			},
		},
		{
			name:      "Not Found",
			accountID: account.ID,
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)
			},
		},
		{
			name:      "Internal Error",
			accountID: account.ID,
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)
			},
		},
		{
			name:      "Invalid ID",
			accountID: 0,
			buildStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorizationHeader(t, request, tokenMaker, authorizationTypeBearer, owner, time.Minute)
			},
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStub(store)

			server := newTestServer(t, store)

			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			tc.setupAuth(t, req, server.tokenMaker)

			require.NoError(t, err)
			server.router.ServeHTTP(recorder, req)
			tc.checkResponse(t, recorder)
		},
		)
	}

}

func randomAccount(owner string) db.Account {
	return db.Account{
		ID:        util.RandomInt(1, 1000),
		Owner:     owner,
		Balance:   util.RandomMoney(),
		Currency:  util.RandomCurrency(),
		CreatedAt: time.Now(),
	}
}
func randomAccountWithCurrency(owner string, currency string) db.Account {
	return db.Account{
		ID:        util.RandomInt(1, 1000),
		Owner:     owner,
		Balance:   util.RandomMoney(),
		Currency:  currency,
		CreatedAt: time.Now(),
	}
}

func randomCurrency() string {
	return util.RandomChoice[string](util.ValidCurrencies)
}

func randomAccountZeroBalance(owner string) db.Account {
	return db.Account{
		ID:        util.RandomInt(1, 1000),
		Owner:     owner,
		Balance:   util.RandomMoney(),
		Currency:  util.RandomCurrency(),
		CreatedAt: time.Now(),
	}
}

func requireMatchAccount(t *testing.T, body *bytes.Buffer, account db.Account) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var getAccount db.Account
	err = json.Unmarshal(data, &getAccount)
	if getAccount.CreatedAt.Equal(account.CreatedAt) {
		getAccount.CreatedAt = account.CreatedAt
	}
	require.NoError(t, err)
	require.Equal(t, getAccount, account)
}

func requireMatchAccounts(t *testing.T, body *bytes.Buffer, accounts []db.Account) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var getAccounts []db.Account
	err = json.Unmarshal(data, &getAccounts)

	require.NoError(t, err)
	require.Equal(t, len(getAccounts), len(accounts))

	for index := 0; index < len(accounts); index++ {
		getAccount, account := getAccounts[index], accounts[index]
		if getAccount.CreatedAt.Equal(account.CreatedAt) {
			getAccount.CreatedAt = account.CreatedAt
		}
		require.NoError(t, err)
		require.Equal(t, getAccount, account)
	}

}
