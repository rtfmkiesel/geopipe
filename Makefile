build: clean
	go build

linux: clean
	GOOS=linux go build

windows: clean
	GOOS=windows go build

all: clean linux windows

clean:
	rm -f geopipe
	rm -f geopipe.exe