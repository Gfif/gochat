package main

import (
	"fmt"
	"github.com/jasocox/figo"
)

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
}

func (h *History) Get() string {
	res := ""
	for el := h.queue.Front(); el != nil; el = h.queue.Next(el) {
		res += fmt.Sprint(el.Value) + FAKE_NEWLINE
	}

	return res + "\n"
}
