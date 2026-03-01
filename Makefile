build:
	go build cmd/main.go
docker-build:
	docker build . -t conops
docker-start:
	docker run --name conops -it --mount type=bind,src=$(CURDIR)/config,dst=/app/config --detach --restart unless-stopped -u 1000:1000 conops
docker-stop:
	docker stop conops
docker-clean:
	docker container rm conops
reboot:
	docker stop conops
	docker container rm conops
	docker build . -t conops
	docker run --name conops -it --mount type=bind,src=$(CURDIR)/config,dst=/app/config --detach --restart unless-stopped -u 1000:1000 conops
