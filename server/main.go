package main

import (
	"flag"
	"github.com/deckarep/golang-set"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	BUF_SIZE     = 100
	FAKE_NEWLINE = "\\\\"
)

var (
	listen = flag.String("b", ":1991", "bind")
)

var (
	hist  = NewHistory(10)
	chans = mapset.NewSet()
	users = mapset.NewSet()
)

func prepareUsersList() string {
	res := ""
	for user := range users.Iter() {
		str, _ := user.(string)
		res += str + FAKE_NEWLINE
	}
	return res + "\n"
}

func isUserExists(user string) bool {
	for u := range users.Iter() {
		str, _ := u.(string)
		if user == str {
			return true
		}
	}
	return false
}

func main() {
	flag.Parse()

	log.Infof("Start server on %s", *listen)
	ln, err := net.Listen("tcp", *listen)
	defer func() {
		if err := ln.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	quit := make(chan int)
	connChan := make(chan net.Conn)
	broad := make(chan string)

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT)

		s := <-sigc
		log.Info(s.String() + " captured. Closing...")
		close(quit)
	}()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.WithError(err).Error("Failed to accept connection")
				continue
			}
			connChan <- conn
		}
	}()

	for {
		select {
		case conn := <-connChan:
			wg.Add(1)
			go NewChatConnection(conn, broad, quit).Handle(&wg)
		case msg := <-broad:
			log.Print(msg)
			hist.Add(msg)
			for c := range chans.Iter() {
				ch, _ := c.(chan string)
				// clear line first
				msg = "\x1b[2K" + msg
				ch <- msg
			}
		case <-quit:
			wg.Wait()
			return
		}
	}
}
