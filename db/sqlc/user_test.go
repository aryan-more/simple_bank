package db

import (
	"context"
	"testing"
	"time"

	"github.com/aryan-more/simple_bank/util"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func createRandomUser(t *testing.T) User {
	hashedPassword, err := util.HashPassword(util.RandomString(8))
	require.NoError(t, err)
	username := util.RandomOwner()
	arg := CreateUserParams{
		FullName:       username,
		Username:       username,
		HashedPassword: hashedPassword,
		Email:          util.RandomEmail(username),
	}
	user, err := testQueries.CreateUser(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.HashedPassword, user.HashedPassword)
	require.Equal(t, arg.Username, user.Username)

	require.True(t, user.PasswordChangedAt.IsZero())
	require.NotZero(t, user.CreatedAt)

	return user

}

func TestGetUser(t *testing.T) {
	randomUser := createRandomUser(t)
	fetchedUser, err := testQueries.GetUser(context.Background(), randomUser.Username)
	require.NoError(t, err)
	require.NotEmpty(t, fetchedUser)

	require.Equal(t, randomUser.Email, fetchedUser.Email)
	require.Equal(t, randomUser.FullName, fetchedUser.FullName)
	require.Equal(t, randomUser.HashedPassword, fetchedUser.HashedPassword)
	require.Equal(t, randomUser.Username, fetchedUser.Username)

	require.WithinDuration(t, randomUser.CreatedAt, fetchedUser.CreatedAt, time.Second)
	require.WithinDuration(t, randomUser.PasswordChangedAt, fetchedUser.PasswordChangedAt, time.Second)

}
