.PHONY: up down dev migrate seed logs build test

up:
	docker compose up -d --build

down:
	docker compose down

dev:
	docker compose -f compose.yml -f compose.dev.yml down -v
	docker compose -f compose.yml -f compose.dev.yml up -d --build db redis
	docker compose -f compose.yml -f compose.dev.yml run --rm migrate
	docker compose -f compose.yml -f compose.dev.yml exec db sh -c 'psql -U "$$POSTGRES_USER" -d "$${POSTGRES_DB:-tt_tracker}" -f /seed/dev_positions.sql'
	docker compose -f compose.yml -f compose.dev.yml up --build -t 3

migrate:
	docker compose run --rm migrate

seed:
	docker compose exec db sh -c 'psql -U "$$POSTGRES_USER" -d "$${POSTGRES_DB:-tt_tracker}" -f /seed/dev_positions.sql'

logs:
	docker compose logs -f

build:
	docker compose build

test:
	go test ./shared/... ./collector/... ./api/...
psql:
	docker compose exec db sh -c 'psql -U "$$POSTGRES_USER" -d "$${POSTGRES_DB:-tt_tracker}"'
