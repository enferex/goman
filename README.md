### goman: Man page parser written in Go

#### What
goman is a man page parsing library.  This tool takes as input a man page that
is gzip (DEFLATE) compressed.  The output is a ManPage object that can be used
however you so choose.

#### Using
Use the _go_ utility to download, build, and install this package:
```
go get github.com/enferex/goman
```

#### Testing
A sample compressed man page is provided: test.1.gz.  The unit test
_goman_test.go_ uses this file:
```
go test goman
```

#### Contact
mattdavis9@gmail.com (enferex)   
