.PHONY: build all run clean

BIN:= fastglue-example.bin

build:
	go build -o ${BIN}

run:
	./${BIN}

clean:
	go clean
	- rm -f ${BIN}

all: clean build run
