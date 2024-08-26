define remove_images:
	docker rmi myapp-ui myapp-service
endef
up:
	docker compose down
	$(remove_images)
	docker compose up -d
down:
	docker compose down
	$(remove_images)