build_amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/pvt-to-jpg-amd64-linux app.go