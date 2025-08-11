.PHONY: up down restart logs psql redis-cli migrate clean generate

# Localda barcha servislarga start berish
up:
	docker-compose up -d --build

# Barcha servislarga stop berish
down:
	docker-compose down

# Restart qilish
restart:
	docker-compose down
	docker-compose up -d --build

# Loglarni ko‘rish
logs:
	docker-compose logs -f

# Postgres CLI
psql:
	docker exec -it my_postgres psql -U postgres -d authservice

# Redis CLI
redis-cli:
	docker exec -it my_redis redis-cli -a 1234

# Migratsiya (birinchi versiya uchun)
migrate:
	docker exec -i my_postgres psql -U postgres -d authservice < ./migrations/postgres/0001_create_user_table.up.sql

# Barcha containerlarni tozalash
clean:
	-docker rm -f my_postgres my_redis go_app_container gotenberg || true
	-docker volume rm $$(docker volume ls -q) || true

# Swagger docs generatsiya
generate:
	swag init -g api/router.go -o api/docs
