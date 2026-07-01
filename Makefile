.PHONY: up down run tidy seed

up:
	docker-compose up -d

down:
	docker-compose down

tidy:
	go mod tidy

run:
	go run ./cmd/server

# Creates one central_admin user: admin@freshtrack.local / admin123

seed:
	docker exec -i freshtrack-postgres psql -U freshtrack -d freshtrack -c \
	"INSERT INTO users (email, password_hash, role) VALUES ('admin@freshtrack.local', '\$$2b\$$10\$$2MsBkUU2en8mWVQTdliIEOjrlSWGTdK195U/2SuvqveJ1T7t6ejMe', 'central_admin') ON CONFLICT (email) DO NOTHING;"
