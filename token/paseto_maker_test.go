package token

import (
	"testing"
	"time"

	"github.com/aryan-more/simple_bank/util"
	"github.com/stretchr/testify/require"
)

func TestPasteoShortSecret(t *testing.T) {
	_, err := NewPasetoMaker("short")
	require.Error(t, err)
}

func TestPasetoTokenMaker(t *testing.T) {

	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomOwner()
	duration := time.Duration(time.Minute)

	issuedAt := time.Now()
	expiredAt := issuedAt.Add(time.Minute)

	token, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	payload, err := maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	require.NotZero(t, payload.ID)
	require.Equal(t, username, payload.Username)
	require.WithinDuration(t, expiredAt, payload.ExpireAt, time.Second)
	require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)

}

func TestExpiredPasetoToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	token, err := maker.CreateToken(util.RandomOwner(), -time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	payload, err := maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Nil(t, payload)

}
