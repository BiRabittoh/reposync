build:
	go build -o sync main.go

run: build
	./sync
