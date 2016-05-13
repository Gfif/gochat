package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"strings"
)

var (
	user = flag.String("u", "", "user name")
	serv = flag.String("s", "localhost:1991", "server address")
)

const (
	CMD_EXT = "bye="
	CMD_REG = "reg="
	CMD_MSG = "msg="
	CMD_LST = "list="
	CMD_ERR = "error="
)

const (
	CLT_CMD_EXT = "/bye"
	CLT_CMD_LST = "/list"
)

const FAKE_NEWLINE = "\\\\"

type Client struct {
	net.Conn
}

func NewClient(server string) *Client {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		log.Fatal(err)
	}
	return &Client{conn}
}

func (c *Client) RunReader() {
	for {
		r := bufio.NewReader(c.Conn)
		text, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Info("Server unexpectedly closed. Please, try to connect again.")
				os.Exit(1)
			}
			log.Fatal(err)
		}

		if strings.HasPrefix(text, CMD_ERR) {
			log.Fatal(errors.New(text[6:]))
		}

		fmt.Print(strings.Replace(text, FAKE_NEWLINE, "\n", -1))
	}
}

func (c *Client) RunWriter() {
	for {
		r := bufio.NewReader(os.Stdin)
		text, _ := r.ReadString('\n')
		switch strings.Replace(text, "\n", "", -1) {
		case CLT_CMD_LST:
			c.Write(CMD_LST)
		case CLT_CMD_EXT:
			c.Write(CMD_EXT)
			return
		default:
			c.WriteMsg(text)
		}
	}
}

func (c *Client) Write(str string) {
	_, err := c.Conn.Write([]byte(str + "\n"))
	if err != nil {
		log.WithError(err).Error("Failed to send data to server")
	}
}

func (c *Client) Register() {
	c.Write(CMD_REG + *user)
}

func (c *Client) WriteMsg(msg string) {
	c.Write(CMD_MSG + msg)

}

func main() {
	flag.Parse()

	c := NewClient(*serv)
	defer c.Close()

	c.Register()

	go c.RunReader()
	c.RunWriter()
}
