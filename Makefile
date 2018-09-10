ldflags = '-extldflags "-static"'

build:
	go build -o zbundle -ldflags $(ldflags)
