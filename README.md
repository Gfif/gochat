GoChat
======

Simple multithread chat

###Requirements###

``` sh
go get github.com/sirupsen/logrus
go get github.com/deckarep/golang-set
go get github.com/jasocox/figo
```

###Build###

``` sh
cd server; go build; cd -
cd client; go build; cd -
```

###Running server###

``` sh
./server/server -b :1991
```

###Running client###

``` sh
./client/client -s localhost:1991 -u mynick
```


