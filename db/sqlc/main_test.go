package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/aryan-more/simple_bank/util"
	_ "github.com/lib/pq"
)

var testQueries *Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatalln("Cannot load config:", err)
	}
	fmt.Println("Starting")
	testDB, err = sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(testDB)

	os.Exit(m.Run())
}
