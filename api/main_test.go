package api

import (
	"os"
	"testing"
	"time"

	db "github.com/aryan-more/simple_bank/db/sqlc"
	"github.com/aryan-more/simple_bank/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store) *Server {

	config := util.Config{
		TokenKey:   util.RandomString(32),
		AccessTime: time.Minute,
	}
	server, err := NewServer(store, config)
	require.NoError(t, err)

	return server

}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
