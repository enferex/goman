APP=goman
SRC=$(APP).go

all: $(APP)

$(APP): $(SRC)
	go build goman.go

.PHONY:test
test: $(APP)
	./$(APP) -f /usr/share/man/man1/7zr.1.gz -d

.PHONY:fmt
fmt: $(SRC)
	go fmt $^
