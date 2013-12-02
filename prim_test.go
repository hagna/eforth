package eforth

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestDoLIST(t *testing.T) {
	f := NewForth()
	f.IP = 99
	binary.LittleEndian.PutUint16(f.Memory[23:], 0xdead)
	f.Push(23) // this is the return address
	f.doLIST()
	b := binary.LittleEndian.Uint16(f.Memory[f.RP:])
	if b != 99 {
		t.Fatal("b should be 99 and not", b)
	}
}

func TestRx(t *testing.T) {
	f := NewForth()
	f.B_IO()
	f.input = bufio.NewReader(strings.NewReader("t"))
	f.Q_RX()
	tf := f.Pop()
	c := f.Pop()
	if c != 't' {
		t.Fatal("c ought to be t not", c)
	}
	if tf != 0xffff {
		t.Fatal("t ought to be -1 not", t)
	}
}

func TestRx2(t *testing.T) {
	f := NewForth()
	f.B_IO()
	f.input = bufio.NewReader(strings.NewReader(""))
	f.Q_RX()
	tf := f.Pop()
	c := f.Pop()
	if c != 0 {
		t.Fatal("c ought to be t not", c)
	}
	if tf != 0 {
		t.Fatal("t ought to be -1 not", t)
	}
}

func TestDoList(t *testing.T) {
	f := NewForth()
	f.B_IO()
	f.IP = 30
	binary.LittleEndian.PutUint16(f.Memory[f.IP:], 0xdead)
	f.doLIT()
	tos := f.Pop()
	if tos != 0xdead {
		t.Fatal("should be dead but tos is", tos)
	}
	if f.IP != 32 {
		t.Fatal("IP should have advanced to", 32)
	}
}

func TestTx(t *testing.T) {
	f := NewForth()
	f.B_IO()
	s := new(bytes.Buffer)
	f.output = bufio.NewWriter(s)
	f.Push(99)
	f.B_TX()
	val := s.String()
	if val != "c" {
		t.Fatal("val should be c and not", val)
	}
}
