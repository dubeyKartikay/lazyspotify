
run:
	go run ./cmd/lazyspotify/main.go

build:
	mkdir -p target
	go build -o target/lazyspotify ./cmd/lazyspotify/main.go 
