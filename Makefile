define remove_containers
endef

up:
	docker compose down
	docker compose up -d
down:
	docker compose down