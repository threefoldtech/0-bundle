# make file to build zbundle with correct name, and build flags

ldflags = '-extldflags "-static"'

build:
	go build -o zbundle -ldflags $(ldflags)
