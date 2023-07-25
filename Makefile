createdb:
	createdb --username=postgres --owner=postgres simple_bank
	
createdb-docker:
	docker exec -it postgres createdb --username=root --owner=root simple_bank

dropdb:
	dropdb -U postgres simple_bank

dropdb-docker:
	docker exec -it postgres dropdb -U root simple_bank

migrateup:
	migrate -path db/migration -database "postgresql://postgres:P35Bxzz6K@localhost:5432/simple_bank?sslmode=disable" -verbose up 

migrateup-docker:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up 

migrateup1:
	migrate -path db/migration -database "postgresql://postgres:P35Bxzz6K@localhost:5432/simple_bank?sslmode=disable" -verbose up 1

migrateup-docker1:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose up 1

migratedown:
	migrate -path db/migration -database "postgresql://postgres:P35Bxzz6K@localhost:5432/simple_bank?sslmode=disable" -verbose down 

migratedown-docker:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down 

migratedown1:
	migrate -path db/migration -database "postgresql://postgres:P35Bxzz6K@localhost:5432/simple_bank?sslmode=disable" -verbose down 1

migratedown-docker1:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable" -verbose down 1

test:
	go test -v -cover ./...

server:
	go run main.go

mock:
	mockgen -destination db/mock/store.go -package mockdb github.com/aryan-more/simple_bank/db/sqlc Store 

sqlc:
	sqlc generate

postgres:
	docker run --name postgres --network bank-network -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=P35Bxzz6K -d postgres:15.3-alpine

rundocker:
	docker run --name simplebank --network bank-network -p 8080:8080 -e GIN_MODE=release -e DB_URL=postgresql://postgres:P35Bxzz6K@postgres:5432/simple_bank?sslmode=disable

recreate_compose:
	docker compose down
	docker rmi simple_bank-api
	docker compose up



.PHONY: createdb createdb-docker dropdb dropdb-docker migrateup migrateup-docker migratedown migratedown-docker mock migrateup1 migrateup-docker1 migratedown1 migratedown-docker1