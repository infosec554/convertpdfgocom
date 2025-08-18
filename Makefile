.PHONY: up down restart logs psql redis-cli migrate clean generate

up:
	docker-compose up -d --build

down:
	docker-compose down

restart:
	docker-compose down
	docker-compose up -d --build

logs:
	docker-compose logs -f

psql:
	docker exec -it my_postgres psql -U postgres -d authservice

redis-cli:
	docker exec -it my_redis redis-cli -a 1234

migrate:
	docker exec -i my_postgres psql -U postgres -d authservice < ./migrations/postgres/0001_create_user_table.up.sql

clean:
	-docker rm -f my_postgres my_redis go_app_container gotenberg || true
	-docker volume rm $$(docker volume ls -q) || true

generate:
	swag init -g api/router.go -o api/docs
