define remove_containers
	docker stop $(docker ps -aqf "name=myapp-service") || true && docker rm $(docker ps -aqf "name=myapp-service") || true
	docker stop $(docker ps -aqf "name=myapp-ui") || true && docker rm $(docker ps -aqf "name=myapp-ui")  || true
endef

up:
	$(remove_containers)
	docker compose up -d
down:
	$(remove_containers)