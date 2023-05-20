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

.PHONY: createdb createdb-docker dropdb dropdb-docker migrateup migrateup-docker migratedown migratedown-docker mock