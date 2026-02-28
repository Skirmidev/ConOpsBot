build:
	go build cmd/main.go
docker-build:
	docker build . -t conops
docker-start:
    docker run --name conops -it --mount type=bind,src=$(pwd)/config,dst=/app/config --detach --restart unless-stopped conops
docker-stop:
	docker stop conops