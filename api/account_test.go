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
	"github.com/aryan-more/simple_bank/util"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestListAccountAPI(t *testing.T) {

	n := 5
	accounts := make([]db.Account, n)
	for index := 0; index < n; index++ {
		accounts[index] = randomAccount()
	}

	type Query struct {
		pageID   int
		pageSize int
	}

	testcase := []struct {
		name          string
		query         Query
		buildStub     func(store *mockdb.MockStore)
		responseCheck func(recorder *httptest.ResponseRecorder)
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
				}

				store.EXPECT().ListAccounts(gomock.Any(), gomock.Eq(arg)).Times(1).Return(accounts, nil)
			},
			responseCheck: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireMatchAccounts(t, recorder.Body, accounts)
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
			responseCheck: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
			responseCheck: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
			responseCheck: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
				server := NewServer(store)

				url := fmt.Sprintf("/accounts")
				req, err := http.NewRequest(http.MethodGet, url, nil)
				require.NoError(t, err)

				q := req.URL.Query()
				q.Add("page_id", fmt.Sprintf("%d", tc.query.pageID))
				q.Add("page_size", fmt.Sprintf("%d", tc.query.pageSize))
				req.URL.RawQuery = q.Encode()

				recorder := httptest.NewRecorder()
				server.router.ServeHTTP(recorder, req)
				tc.responseCheck(recorder)

			},
		)
	}

}

func TestCreateAccount(t *testing.T) {
	account := randomAccountZeroBalance()
	testcase := []struct {
		name          string
		body          gin.H
		buildStub     func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
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
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireMatchAccount(t, recorder.Body, account)
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
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
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
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
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
				server := NewServer(store)

				data, err := json.Marshal(tc.body)
				require.NoError(t, err)

				recorder := httptest.NewRecorder()
				url := "/accounts"
				req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))

				require.NoError(t, err)
				server.router.ServeHTTP(recorder, req)

				tc.checkResponse(recorder)
			},
		)
	}

}

func TestGetAccount(t *testing.T) {
	account := randomAccount()

	testcase := []struct {
		name          string
		accountID     int64
		buildStub     func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
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
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStub(store)

			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			req, err := http.NewRequest(http.MethodGet, url, nil)

			require.NoError(t, err)
			server.router.ServeHTTP(recorder, req)
			tc.checkResponse(t, recorder)
		},
		)
	}

}

func randomAccount() db.Account {
	return db.Account{
		ID:        util.RandomInt(1, 1000),
		Owner:     util.RandomOwner(),
		Balance:   util.RandomMoney(),
		Currency:  util.RandomCurrency(),
		CreatedAt: time.Now(),
	}
}
func randomAccountZeroBalance() db.Account {
	return db.Account{
		ID:        util.RandomInt(1, 1000),
		Owner:     util.RandomOwner(),
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
