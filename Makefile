up:
	(docker compose down || true) && (docker rmi myapp-ui myapp-service || true) && docker compose up -d
down:
	(docker compose down || true) && (docker rmi myapp-ui myapp-service || true)