package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"

	"github.com/aryan-more/simple_bank/api"
	db "github.com/aryan-more/simple_bank/db/sqlc"
)

const (
	dbDriver = "postgres"
	dbSource = "postgresql://postgres:P35Bxzz6K@localhost:5432/simple_bank?sslmode=disable"
	address  = "0.0.0.0:8080"
)

func main() {
	conn, err := sql.Open(dbDriver, dbSource)
	// gin.SetMode(gin.ReleaseMode)
	if err != nil {
		log.Fatal("Cannot connect to db:", err)
	} else {
		log.Println("Connected to db")
	}

	store := db.NewStore(conn)

	server := api.NewServer(store)

	err = server.Start(address)
	if err != nil {
		log.Fatal("Cannot Start Server:", err)
	} else {
		log.Panicln("Server Started")
	}
}
