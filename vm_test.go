package eforth

import (
	"fmt"
	"strings"
	"testing"
)

func TestFmt(t *testing.T) {
	f := Forth{}
	fmt.Println("hello world")
	fmt.Println(f.Memory)
}

func TestPushPop(t *testing.T) {
	f := NewForth()
	f.Push(0x100)
	f.IP = f.Pop()
	if f.IP != 0x100 {
		t.Fatal("f.IP should be 0x100 but it is", f.IP)
	}
}

func TestWordPtr(t *testing.T) {
	f := NewForth()
	w := f.WordPtr(10)
	if w != 0 {
		t.Fatal("oughta be 0")
	}
	f.SetWordPtr(10, 100)
	w = f.WordPtr(10)
	if w != 100 {
		t.Fatal("oughta be 100 and not", w)
	}
}

func TestRegLower(t *testing.T) {
	f := NewForth()
	z := f.RegLower(0xdead)
	if z != 0xad {
		t.Fatal("should be ad but got", z)
	}
}

func StrsEquals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestFields(t *testing.T) {
	f := `COLON 1PLUS 1 PLUS
            DOT EXIT
            `
	res := strings.Fields(f)
	if !StrsEquals(res, []string{"COLON", "1PLUS", "1", "PLUS", "DOT", "EXIT"}) {
		t.Fatal(f, "changed to", strings.Fields(f))
	}
}

func TestAddPrim(t *testing.T) {
	f := NewForth()
	a := func() {}
	f.AddPrim("BYE", a)
	code := f.WordPtr(CODEE) // first word in code dictionary
	if code != 1 {
		t.Fatal("ought to have a code word there")
	}
}

func TestAddPrimCall(t *testing.T) {
	f := NewForth()
	called := false
	f.AddPrim("BYE", func() { called = true })
	f.Call("BYE")
	if called != true {
		t.Fatal("didn't call the function")
	}
}

func TestMorePrim(t *testing.T) {
	f := NewForth()
	f.prims = 0
	f.AddPrim("BYE", func() {})
	f.AddPrim("JK", func() {})
	a := f.Addr("JK")
	if a != CODEE+2 {
		t.Fatal("didn't add new code in memory")
	}
	b := f.WordPtr(CODEE + 2)
	if "JK" != f.pcode2word[b] {
		t.Fatal("memory not set to pcode of JK", b)
	}
}

func TestPcode2Prim(t *testing.T) {
	f := NewForth()
	f.prims = 0
	f.AddPrim("BYE", func() {})
	f.AddPrim("JK", func() {})
	a := f.Frompcode(2)
	if a != "JK" {
		t.Fatal("Looking up pcode 2 we got", a, "but should have got JK")
	}

}

func TestAddPrimCall2(t *testing.T) {
	f := NewForth()
	f.IP = 0
	o := f.IP
	f.AddPrim("NEXT", f.NEXT)
	f.Call("NEXT")
	o2 := f.IP
	if o2 <= o {
		t.Fatal("didn't call the function NEXT or NEXT implementation has changed")
	}
}

func TestAddPrimAddr(t *testing.T) {
	f := NewForth()
	f.prims = 0
	a := func() {}
	f.AddPrim("BYE", a)
	if f.Addr("BYE") != CODEE {
		t.Fatal("address of BYE ought to be", CODEE, "and not ", f.Addr("BYE"))
	}
}
