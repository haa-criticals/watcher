proto:
	protoc -I=./app/proto  --go_out=./app/grpc/pb --go_opt=paths=source_relative --go-grpc_out=./app/grpc/pb --go-grpc_opt=paths=source_relative  ./app/proto/*.proto

build:
	go build -o dist/watcher .

run:
	go run main.go

create-gitlab:
	dist/watcher create -i 39489419 -b https://gitlab.com -t "glptt-35ab9be4a7d85e92f1e3a9a7c67eea23f78c0695" -v "PROVIDER=digitalocean" -v "BACKEND_SCHEMA=haacs_gitlab"
