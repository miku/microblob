microblob: cmd/microblob/main.go
	go build -o microblob cmd/microblob/main.go

clean:
	rm -f microblob
