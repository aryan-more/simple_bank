package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestPassword(t *testing.T) {
	password := RandomString(8)
	hashedPassword, err := HashPassword(password)

	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword)

	require.NoError(t, CheckedPassword(password, hashedPassword))
	require.EqualError(t, CheckedPassword(RandomString(10), hashedPassword), bcrypt.ErrMismatchedHashAndPassword.Error())
}
