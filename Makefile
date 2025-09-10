
SHELL := /bin/bash

DC            ?= docker compose
PROFILE       ?= local                    # local yoki prod
COMPOSE_FLAGS ?= --profile $(PROFILE)

APP_SVC       ?= app
PG_SVC        ?= postgres
REDIS_SVC     ?= redis
GOTENBERG_SVC ?= gotenberg

# PG sozlamalari (local profilda compose ichidagi xizmatlar)
PGUSER        ?= postgres
PGPASSWORD    ?= 1234
PGDATABASE    ?= authservice
PGHOST_LOCAL  ?= $(PG_SVC)
PGPORT_LOCAL  ?= 5432
# Prod (RDS) uchun tashqi ulanish (istalgan payt override qilasan)
PGHOST_PROD   ?=
PGPORT_PROD   ?= 5432

# Redis parol localda minimal himoya
REDIS_PASSWORD ?= 1234

# Migrations papka
MIG_DIR       ?= migrations/postgres

# Swag docs manzili (agar kerak bo‘lsa)
SWAG_MAIN     ?= api/router.go
SWAG_OUT      ?= api/docs

.PHONY: up up-local up-prod down restart logs logs-app logs-db logs-redis logs-gotenberg \
        psql psql-rds redis-cli migrate-one migrate-all rollback-one rollback-all \
        db-create-if-missing wait-db clean clean-volumes generate health

# ================================
# Run / Stop
# ================================
up:
	$(DC) $(COMPOSE_FLAGS) up -d --build

up-local:
	$(MAKE) up PROFILE=local

up-prod:
	$(MAKE) up PROFILE=prod

down:
	$(DC) $(COMPOSE_FLAGS) down

restart:
	$(MAKE) down PROFILE=$(PROFILE)
	$(MAKE) up PROFILE=$(PROFILE)

# ================================
# Logs (ajratib ko‘rish qulay)
# ================================
logs:
	$(DC) $(COMPOSE_FLAGS) logs -f

logs-app:
	$(DC) $(COMPOSE_FLAGS) logs -f $(APP_SVC)

logs-db:
	$(DC) $(COMPOSE_FLAGS) logs -f $(PG_SVC)

logs-redis:
	$(DC) $(COMPOSE_FLAGS) logs -f $(REDIS_SVC)

logs-gotenberg:
	$(DC) $(COMPOSE_FLAGS) logs -f $(GOTENBERG_SVC)

# ================================
# PSQL / Redis CLI
# ================================
psql:
	# Compose ichidagi Postgres'ga kirish (local profil uchun)
	$(DC) $(COMPOSE_FLAGS) exec -it $(PG_SVC) psql -U $(PGUSER) -d $(PGDATABASE)

psql-rds:
	# Tashqi RDS'ga bevosita host orqali ulangan holda (PGHOST_PROD ni to‘ldir)
	@[ -n "$(PGHOST_PROD)" ] || (echo "Set PGHOST_PROD for psql-rds" && exit 1)
	PGPASSWORD=$(PGPASSWORD) psql -h $(PGHOST_PROD) -p $(PGPORT_PROD) -U $(PGUSER) -d $(PGDATABASE)

redis-cli:
	$(DC) $(COMPOSE_FLAGS) exec -it $(REDIS_SVC) redis-cli -a $(REDIS_PASSWORD)

# ================================
# DB health / init helpers
# ================================
wait-db:
	# PG tayyor bo‘lguncha kutish (compose healthcheck bor, lekin yana kafolat)
	@echo "Waiting for Postgres to be ready..."
	@for i in {1..30}; do \
		$(DC) $(COMPOSE_FLAGS) exec -T $(PG_SVC) pg_isready -U $(PGUSER) -d $(PGDATABASE) && exit 0; \
		echo "retry $$i"; sleep 2; \
	done; \
	echo "Postgres is not ready" && exit 1

db-create-if-missing:
	# RDS yoki compose ichida DB yo‘q bo‘lsa, yaratadi
	@echo "Ensuring database '$(PGDATABASE)' exists..."
	$(DC) $(COMPOSE_FLAGS) exec -T $(PG_SVC) psql -U $(PGUSER) -v ON_ERROR_STOP=1 -d postgres -c "\
	DO $$ \
	BEGIN \
	   IF NOT EXISTS (SELECT FROM pg_database WHERE datname = '$(PGDATABASE)') THEN \
	      PERFORM dblink_exec('dbname=postgres', 'CREATE DATABASE $(PGDATABASE)'); \
	   END IF; \
	END $$;" 2>/dev/null || \
	$(DC) $(COMPOSE_FLAGS) exec -T $(PG_SVC) psql -U $(PGUSER) -v ON_ERROR_STOP=1 -d postgres -c "CREATE DATABASE $(PGDATABASE);" || true
	@echo "Database check complete."

# ================================
# Migrations (SQL fayllar)
# — 0001_xxx.up.sql, 0002_xxx.up.sql ... tartibida qo‘llanadi
# — rollback uchun .down.sql teskari tartibda
# ================================
migrate-one:
	# Bitta faylni qo‘llash: make migrate-one FILE=.../0001_xxx.up.sql
	@[ -n "$(FILE)" ] || (echo "Usage: make migrate-one FILE=$(MIG_DIR)/0001_xxx.up.sql" && exit 1)
	$(DC) $(COMPOSE_FLAGS) exec -T $(PG_SVC) psql -U $(PGUSER) -d $(PGDATABASE) -v ON_ERROR_STOP=1 -f /dev/stdin < $(FILE)
	@echo "Applied: $(FILE)"

migrate-all: wait-db
	# Barcha *.up.sql fayllarni tartib bilan qo‘llash
	@set -e; \
	shopt -s nullglob; \
	files=($(MIG_DIR)/*_*.up.sql); \
	if [ $$(( ${#files[@]} )) -eq 0 ]; then echo "No .up.sql files in $(MIG_DIR)"; exit 0; fi; \
	for f in $${files[@]}; do \
		echo "Applying $$f"; \
		$(DC) $(COMPOSE_FLAGS) exec -T $(PG_SVC) psql -U $(PGUSER) -d $(PGDATABASE) -v ON_ERROR_STOP=1 -f /dev/stdin < $$f; \
	done; \
	echo "All UP migrations applied."

rollback-one:
	# Bitta DOWN fayl: make rollback-one FILE=.../0001_xxx.down.sql
	@[ -n "$(FILE)" ] || (echo "Usage: make rollback-one FILE=$(MIG_DIR)/0001_xxx.down.sql" && exit 1)
	$(DC) $(COMPOSE_FLAGS) exec -T $(PG_SVC) psql -U $(PGUSER) -d $(PGDATABASE) -v ON_ERROR_STOP=1 -f /dev/stdin < $(FILE)
	@echo "Rolled back: $(FILE)"

rollback-all:
	# Barcha *.down.sql fayllarni TESKARI tartibda (katta raqamdan kichikgacha)
	@set -e; \
	shopt -s nullglob; \
	files=($(ls -1 $(MIG_DIR)/*_*.down.sql | sort -r)); \
	if [ $$(( ${#files[@]} )) -eq 0 ]; then echo "No .down.sql files in $(MIG_DIR)"; exit 0; fi; \
	for f in $${files[@]}; do \
		echo "Rolling back $$f"; \
		$(DC) $(COMPOSE_FLAGS) exec -T $(PG_SVC) psql -U $(PGUSER) -d $(PGDATABASE) -v ON_ERROR_STOP=1 -f /dev/stdin < $$f; \
	done; \
	echo "All DOWN migrations applied."

# Oldingi bitta fayl piping varianti (saqlab qo'ydim)
migrate:
	$(DC) $(COMPOSE_FLAGS) exec -T $(PG_SVC) psql -U $(PGUSER) -d $(PGDATABASE) -v ON_ERROR_STOP=1 -f /dev/stdin < $(MIG_DIR)/0001_create_user_table.up.sql

# ================================
# Health probe (ichki servislar)
# ================================
health:
	@echo "App health:" && curl -sf http://localhost:8080/health || true
	@echo -e "\nGotenberg health (container ichidan):"
	$(DC) $(COMPOSE_FLAGS) exec -T $(GOTENBERG_SVC) wget -qO- http://localhost:3000/health || true

# ================================
# Cleanups
# ================================
clean:
	-$(DC) $(COMPOSE_FLAGS) down || true
	-docker system prune -f || true

clean-volumes:
	-$(DC) $(COMPOSE_FLAGS) down -v || true
	-docker system prune -f || true

# ================================
# Swagger (agar kerak bo‘lsa)
# ================================
generate:
	swag init -g $(SWAG_MAIN) -o $(SWAG_OUT)
