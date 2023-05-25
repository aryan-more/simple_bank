package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	mockdb "github.com/aryan-more/simple_bank/db/mock"
	db "github.com/aryan-more/simple_bank/db/sqlc"
	"github.com/aryan-more/simple_bank/util"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type createUserMatcher struct {
	arg      db.CreateUserParams
	password string
}

// Matches returns whether x is a match.
func (c createUserMatcher) Matches(x interface{}) bool {
	fmt.Println("Match")
	user, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}
	fmt.Println(c.arg)
	fmt.Println(user)

	if err := util.CheckedPassword(c.password, user.HashedPassword); err != nil {
		return false
	}

	c.arg.HashedPassword = user.HashedPassword

	return reflect.DeepEqual(c.arg, user)
}

// String describes what the matcher matches.
func (c createUserMatcher) String() string {
	return fmt.Sprintf("mathches arg %v and password %v", c.arg, c.password)
}

func eqCreateUserMatcher(arg db.CreateUserParams, password string) gomock.Matcher {
	return createUserMatcher{
		arg:      arg,
		password: password,
	}
}

func randomUser(t *testing.T) (user db.User, password string) {
	password = util.RandomString(8)
	hashedPassword, err := util.HashPassword(password)

	require.NoError(t, err)
	user = db.User{
		Username:       util.RandomOwner(),
		FullName:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		Email:          util.RandomEmail(util.RandomOwner()),
	}

	return
}

func TestCreateUserAPI(t *testing.T) {
	user, password := randomUser(t)

	testcase := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateUserParams{
					Username: user.Username,
					FullName: user.FullName,
					Email:    user.Email,
				}
				store.
					EXPECT().
					CreateUser(gomock.Any(), eqCreateUserMatcher(arg, password)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchUser(t, recorder.Body, user)
			},
		},
		{
			name: "Invalid Email",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Username,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Short Password",
			body: gin.H{
				"username":  user.Username,
				"password":  "12345",
				"full_name": user.FullName,
				"email":     user.Username,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Invalid Username",
			body: gin.H{
				"username":  "new@45#",
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Username,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Duplicate",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(
						db.User{},
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
				tc.buildStubs(store)

				server := newTestServer(t, store)

				recorder := httptest.NewRecorder()

				data, err := json.Marshal(tc.body)
				require.NoError(t, err)

				url := "/users"
				request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
				require.NoError(t, err)

				server.router.ServeHTTP(recorder, request)
				tc.checkResponse(recorder)

			},
		)
	}
}

func TestLoginUser(t *testing.T) {

	user, password := randomUser(t)

	testcase := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, response *httptest.ResponseRecorder)
	}{
		{
			name: "Ok",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.Username)).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, response.Code)
				var res logInUserResponse
				err := json.Unmarshal(response.Body.Bytes(), &res)
				require.NoError(t, err)
				data, err := json.Marshal(res.User)
				requireBodyMatchUser(t, bytes.NewBuffer(data), user)

			},
		},

		{
			name: "Username not found",
			body: gin.H{
				"username": util.RandomString(8),
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(1).Return(db.User{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, response.Code)

			},
		},
		{
			name: "Wrong Password",
			body: gin.H{
				"username": user.Username,
				"password": util.RandomString(8),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, response.Code)

			},
		},
		{
			name: "Empty data",
			body: gin.H{
				"username": "",
				"password": "",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, response.Code)

			},
		},
		{
			name: "Wrong Body",
			body: gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.Username)).Times(0)
			},
			checkResponse: func(t *testing.T, response *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, response.Code)

			},
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockdb := mockdb.NewMockStore(ctrl)

			tc.buildStubs(mockdb)
			server := newTestServer(t, mockdb)

			url := "/users/login"

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, req)
			tc.checkResponse(t, recorder)
		})

	}
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, user db.User) {
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	var responseUser db.User
	err = json.Unmarshal(data, &responseUser)
	require.NoError(t, err)
	require.Equal(t, user.Username, responseUser.Username)
	require.Equal(t, user.FullName, responseUser.FullName)
	require.Equal(t, user.Email, responseUser.Email)
	require.Equal(t, user.CreatedAt.Local(), responseUser.CreatedAt.Local())
	require.Equal(t, user.PasswordChangedAt.Local(), responseUser.PasswordChangedAt.Local())
	require.Empty(t, responseUser.HashedPassword)

}
