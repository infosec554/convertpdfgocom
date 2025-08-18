# Docker Compose xizmatlari

## PostgreSQL
- Container nomi: my_postgres
- Ichki port: 5432
- Host port: 5433 (faqat tashqi ulanish uchun)
- `.env` faylda POSTGRES_HOST=my_postgres, POSTGRES_PORT=5432

## Redis
- Container nomi: my_redis
- Ichki port: 6379
- Host port: 6380 (faqat tashqi ulanish uchun)
- `.env` faylda REDIS_HOST=my_redis, REDIS_PORT=6379

## Gotenberg
- Container nomi: gotenberg
- Ichki port: 3000
- `.env` faylda GOTENBERG_URL=http://gotenberg:3000
