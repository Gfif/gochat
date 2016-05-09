package main

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
)

var (
	ERROR_ALREADY_EXISTS     = errors.New("user already exists")
	ERROR_ALREADY_REGISTERED = errors.New("you already registered")
)

type ChatConnection struct {
	conn     net.Conn
	nick     string
	quit     chan int
	broad    chan string
	readChan chan string
	logger   *log.Entry
}

func NewChatConnection(conn net.Conn, broad chan string, quit chan int) *ChatConnection {
	return &ChatConnection{
		conn:     conn,
		quit:     quit,
		broad:    broad,
		readChan: make(chan string),
		logger:   log.WithFields(log.Fields{}),
	}
}

func (cc *ChatConnection) isRegistered() bool {
	if cc.nick == "" {
		cc.Write("error=not registered\n")
		return false
	}
	return true
}

func (cc *ChatConnection) Listen() {
	var cmd string
	for cmd != "bye=\n" {
		// TODO: read all
		b := make([]byte, BUF_SIZE)
		n, err := cc.conn.Read(b)
		cmd = string(b[:n])
		if err != nil {
			if err == io.EOF {
				cmd = "bye=\n"
			} else {
				cc.logger.WithError(err).Error("OMG")
			}
		}
		cc.readChan <- cmd
	}
}

func (cc *ChatConnection) Write(s string) {
	if _, err := cc.conn.Write([]byte(s)); err != nil {
		cc.logger.WithError(err).Error()
	}
}

func (cc *ChatConnection) WriteError(err error) {
	cc.Write("error=" + err.Error() + "\n")
}

func (cc *ChatConnection) Exec(cmd *Command) bool {
	switch cmd.commandType {
	case CMD_REG:
		if cc.nick != "" {
			cc.WriteError(ERROR_ALREADY_REGISTERED)
			return false
		}
		if isUserExists(cmd.value) {
			cc.WriteError(ERROR_ALREADY_EXISTS)
			return false
		}
		cc.nick = cmd.value
		cc.logger = cc.logger.WithField("nick", cc.nick)
		// add user's nick to users list
		users.Add(cc.nick)
		// write history to client
		cc.Write(hist.Get())
	case CMD_MSG:
		if cc.isRegistered() {
			cc.broad <- cc.nick + ": " + cmd.value
		}
	case CMD_LST:
		if cc.isRegistered() {
			cc.Write(prepareUsersList())
		}
	case CMD_EXT:
		users.Remove(cc.nick)
		cc.logger.Print("Disconnected")
		return true
	}

	return false
}

func (cc *ChatConnection) Handle(wg *sync.WaitGroup) {
	defer wg.Done()
	defer cc.conn.Close()

	// create and register input channel
	out := make(chan string)
	chans.Add(out)
	defer chans.Remove(out)

	cc.logger = cc.logger.WithField("raddr", cc.conn.RemoteAddr().String())
	cc.logger.Info("Connected")

	// run cc.readChan writer
	go cc.Listen()

	for {
		select {
		case <-cc.quit:
			cc.logger.Print("Closing connection")
			return
		case cmdString := <-cc.readChan:
			cmd, err := ParseCommand(cmdString)
			if err != nil {
				cc.logger.WithError(err).Error("Failed to parse command")
				continue
			}
			if cc.Exec(cmd) {
				return
			}
		case outMsg := <-out:
			cc.Write(outMsg + "\n")
		}
	}
}
