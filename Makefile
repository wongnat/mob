build:
	mkdir bin
	go build -o ./bin/client client/client.go
	go build -o ./bin/tracker tracker/tracker.go

run-client:
	./bin/client

run-tracker:
	./bin/tracker 1234

clean:
	rm -rf bin
