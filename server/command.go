package main

import (
	"errors"
	"strings"
)

const (
	CMD_EXT = "bye"
	CMD_REG = "reg"
	CMD_MSG = "msg"
	CMD_LST = "list"
)

var ERROR_WRONG_COMMAND = errors.New("wrong command format")

type Command struct {
	commandType string
	value       string
}

func ParseCommand(cmd string) (*Command, error) {
	parts := strings.Split(cmd, "=")
	if len(parts) != 2 {
		return nil, ERROR_WRONG_COMMAND
	}
	// TODO: validate commandType
	return &Command{parts[0], strings.Replace(parts[1], "\n", "", -1)}, nil
}
