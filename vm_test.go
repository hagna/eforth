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
	f.CallFn("BYE")
	if called != true {
		t.Fatal("didn't call the function")
	}
}

func TestMorePrim(t *testing.T) {
	f := NewForth()
	f.prims = 0
	f.AddPrim("BYE", func() {})
	f.AddPrim("JK", func() {})
	a, _ := f.Addr("JK")
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
	f.CallFn("NEXT")
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
    aa, _ := f.Addr("BYE")
	if aa != CODEE {
		t.Fatal("address of BYE ought to be", CODEE, "and not ", aa)
	}
}

// Adding a colon word creates a new word named after the first string after the colon
func TestAddColon(t *testing.T) {
    f := NewForth()
    f.AddWord(": nop ;")
    aa, _ := f.Addr("nop")
    if aa == 0 {
        t.Fatal("nop should have an address", aa)
    }
}

func Uint16sEqual(a, b []uint16) bool {
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


// A colon word also ought to go in the code dictionary correctly
func TestAddColonLst(t *testing.T) {
    f := NewForth()
    f.AddWord(": nop ;")
    good := []uint16{}
    for _, v := range []string{"call", ":", ";"} {
        aa, _ := f.Addr(v)
        good = append(good, aa)
    }
    start, _ := f.Addr("nop")
    cmp := []uint16{f.WordPtr(start), f.WordPtr(start+2), f.WordPtr(start+4)}
    if !Uint16sEqual(cmp, good) {
        t.Fatal("code of nop wasn't", good, "but was", cmp)
    }
}

// ought to be able to build definitions on others
func TestAddWords(t *testing.T) {
    f := NewForth()
    f.AddWord(`: nop ;`)
    f.AddWord(`: bar nop ;`)
    good := []uint16{}
    for _, v := range []string{"call", ":", "nop", ";"} {
        aa, _ := f.Addr(v)
        good = append(good, aa)
    }
    start, _ := f.Addr("bar")
    cmp := []uint16{f.WordPtr(start), f.WordPtr(start+2), f.WordPtr(start+4), f.WordPtr(start+6)}
    if !Uint16sEqual(cmp, good) {
        t.Fatal("code of nop wasn't", good, "but was", cmp)
    }
    anop, _ := f.Addr("nop")
    abar, _ := f.Addr("bar")
    if abar - anop != 6 {
        t.Fatal("addresses may overlap nop is at", anop, "and bar is at", abar)
    }

}

// ought to include newlines without messing up the code
func TestAddWordNL(t *testing.T) {
    f := NewForth()
    f.AddWord(`: nop ;`)
    f.AddWord(`: bar 
nop 
;`)
    good := []uint16{}
    for _, v := range []string{"call", ":", "nop", ";"} {
        aa, _ := f.Addr(v)
        good = append(good, aa)
    }
    start, _ := f.Addr("bar")
    cmp := []uint16{f.WordPtr(start), f.WordPtr(start+2), f.WordPtr(start+4), f.WordPtr(start+6)}
    if !Uint16sEqual(cmp, good) {
        t.Fatal("code of nop wasn't", good, "but was", cmp)
    }
}

// ignore comments for now
func TestAddWordComments(t *testing.T) {
    f := NewForth()
    f.AddWord(": nop ( -- ) ;")
    good := []uint16{}
    for _, v := range []string{"call", ":", ";"} {
        aa, _ := f.Addr(v)
        good = append(good, aa)
    }
    start, _ := f.Addr("nop")
    cmp := []uint16{f.WordPtr(start), f.WordPtr(start+2), f.WordPtr(start+4)}
    if !Uint16sEqual(cmp, good) {
        t.Fatal("code of nop wasn't", good, "but was", cmp)
    }
}

func BadAddr(t *testing.T) {
    f := NewForth()
    _, err := f.Addr("noexiste")
    if err == nil {
        t.Fatal("should have thrown an error about a nonexistant word")
    }
}
