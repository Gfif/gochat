package main

import (
	"errors"
	"fmt"
	"github.com/deckarep/golang-set"
	"github.com/jasocox/figo"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const BUF_SIZE = 100

// ------------ Command ----------------------
type Command struct {
	commandType string
	value       string
}

func ParseCommand(cmd string) (*Command, error) {
	parts := strings.Split(cmd, "=")
	if len(parts) != 2 {
		return nil, errors.New("wrong command format")
	}
	// TODO: validate commandType
	return &Command{parts[0], strings.Replace(parts[1], "\n", "", -1)}, nil
}

// -------------- History ---------------------
type History struct {
	queue  figo.Queue
	Len    int
	MaxLen int
}

func NewHistory(length int) *History {
	return &History{figo.New(), 0, length}
}

func (h *History) Add(msg string) {
	if h.Len >= h.MaxLen {
		h.queue.Pop()
		h.queue.Push(msg)
	} else {
		h.queue.Push(msg)
		h.Len++
	}
	log.Printf("%s added. Len: %d, MaxLen: %d", msg, h.Len, h.MaxLen)
}

func (h *History) Get() string {
	res := ""
	for el := h.queue.Front(); el != nil; el = h.queue.Next(el) {
		res += fmt.Sprint(el.Value) + "\n"
	}

	return res
}

func prepareUserList(users mapset.Set) string {
	res := ""
	for user := range users.Iter() {
		str, _ := user.(string)
		res += str + "\n"
	}
	return res
}

// --------------------------------------------

var (
	hist  = NewHistory(10)
	chans = mapset.NewSet()
	users = mapset.NewSet()
)

func listenConn(conn net.Conn, readChan chan string) {
	var cmd string
	for cmd != "bye=\n" {
		// TODO: read all
		b := make([]byte, BUF_SIZE)
		n, err := conn.Read(b)
		cmd = string(b[:n])
		if err != nil {
			if err == io.EOF {
				cmd = "bye=\n"
			} else {
				log.WithError(err).Error("OMG")
			}
		}
		readChan <- cmd
	}
}

func handleConnection(conn net.Conn, wg *sync.WaitGroup, quit chan int, broad chan string) {
	defer wg.Done()
	defer conn.Close()

	// create and register input channel
	in := make(chan string)
	chans.Add(in)
	defer chans.Remove(in)

	logger := log.WithField("raddr", conn.RemoteAddr().String())
	logger.Info("Connected")

	if _, err := conn.Write([]byte(hist.Get())); err != nil {
		logger.WithError(err).Error()
	}

	var nick string
	readChan := make(chan string)
	go listenConn(conn, readChan)

	for {
		select {
		case <-quit:
			logger.Print("Closing connection")
			return
		case cmdString := <-readChan:
			c, err := ParseCommand(cmdString)
			if err != nil {
				logger.WithError(err).Error("Failed to parse command")
				continue
			}
			if c.commandType == "nick" {
				nick = c.value
				logger = logger.WithField("nick", nick)
				// add user's nick to users list
				users.Add(nick)
				defer users.Remove(nick)
			} else if c.commandType == "msg" {
				broad <- nick + ": " + c.value
			} else if c.commandType == "bye" {
				logger.Print(nick + " disconnected")
				return
			} else if c.commandType == "list" {
				if _, err := conn.Write([]byte(prepareUserList(users))); err != nil {
					logger.WithError(err).Error()
				}
			}
		case inMsg := <-in:
			if _, err := conn.Write([]byte(inMsg + "\n")); err != nil {
				logger.WithError(err).Error()
			}
		}
	}
}

func main() {
	ln, err := net.Listen("tcp", ":1991")
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case conn := <-connChan:
				wg.Add(1)
				go handleConnection(conn, &wg, quit, broad)
			case msg := <-broad:
				log.Print(msg)
				hist.Add(msg)
				for c := range chans.Iter() {
					ch, _ := c.(chan string)
					ch <- msg
				}
			case <-quit:
				return
			}
		}
	}()

	wg.Wait()
}
