APP=goman
SRC=$(APP).go

all: $(APP)

$(APP): $(SRC)
	go build goman.go
