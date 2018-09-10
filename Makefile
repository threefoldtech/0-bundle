# make file to build zbundle with correct name, and build flags

ldflags = '-w -s'

build:
	go build -o zbundle -ldflags $(ldflags)
