package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"

	"github.com/aryan-more/simple_bank/api"
	db "github.com/aryan-more/simple_bank/db/sqlc"
	"github.com/aryan-more/simple_bank/util"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatalln("Cannot load config:", err)
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	// gin.SetMode(gin.ReleaseMode)
	if err != nil {
		log.Fatal("Cannot connect to db:", err)
	} else {
		log.Println("Connected to db")
	}

	store := db.NewStore(conn)

	server, err := api.NewServer(store, config)
	if err != nil {
		log.Fatalf("Failed to create server %s", err.Error())
	}

	err = server.Start(config.Address)
	if err != nil {
		log.Fatal("Cannot Start Server:", err)
	} else {
		log.Panicln("Server Started")
	}
}
