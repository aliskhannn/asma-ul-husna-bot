# Build and start all Docker services
docker-up:
	docker compose up --build

# Stop and remove all Docker services and volumes
docker-down:
	docker compose down -v