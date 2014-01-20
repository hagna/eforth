package eforth

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestDoLIST(t *testing.T) {
	f := New(nil, nil)
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
	f := New(nil, nil)
	f.Input = strings.NewReader("t")
	f._B_IO()
	f._Q_RX()
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
	f := New(nil, nil)
	f.Input = strings.NewReader("")
	f._B_IO()
	f._Q_RX()
	tf := f.Pop()
	c := f.Pop()
	if c != 0 {
		t.Fatal("c ought to be t not", c)
	}
	if tf != 0 {
		t.Fatal("t ought to be -1 not", t)
	}
}

func TestDoLit(t *testing.T) {
	f := New(nil, nil)
	f._B_IO()
	f.IP = 30
	binary.LittleEndian.PutUint16(f.Memory[f.IP:], 0xdead)
	f.doLIT()
	tos := f.Pop()
	if tos != 0xdead {
		t.Fatal("should be dead but tos is", tos)
	}
	if f.IP != 34 {
		t.Fatal("IP should have advanced to 34 but was", f.IP)
	}
}

func TestTx(t *testing.T) {
	f := New(nil, nil)
	s := new(bytes.Buffer)
	f.Output = s
	f._B_IO()
	f.Push(99)
	f._B_TX()
	val := s.String()
	if val != "c" {
		t.Fatal("val should be c and not", val)
	}
}
