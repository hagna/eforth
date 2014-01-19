package eforth

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestWordFromASM(t *testing.T) {
	f := New(nil, nil)
	asm := `
;   doTEN ( -- a )
;		testing routine

		$COLON	COMPO+5,'doTEN',DOTEN
		DW	DOLIT,99,EXIT
`
	f.WordFromASM(asm)
	RunWord("doTEN", f, t)
	a := f.Pop()
	if a != 99 {
		t.Fatal(asm, "didn't leave 99 on the stack left", a)
	}
	v := f.WordPtr(f.LAST - 2)
	t.Log("before", dumpmem(f, v, 10))
	v = f.WordPtr(v - 2)
	vl := f.WordPtr(v)
	t.Log("v&mask == ", v&0x0f0, vl)
	if vl&0x00f0 != COMPO {
		t.Log("Here is LAST", dumpmem(f, v, 10))
		t.Log("couldn't find bit")
		t.Fail()
	}
}

func TestImedd(t *testing.T) {
	f := New(nil, nil)
	asm := `
;   doTEN ( -- a )
;		testing routine

		$COLON	IMEDD+5,'doTEN',DOTEN
		DW	DOLIT,99,EXIT
`
	f.WordFromASM(asm)
	RunWord("doTEN", f, t)
	a := f.Pop()
	if a != 99 {
		t.Fatal(asm, "didn't leave 99 on the stack left", a)
	}
	v := f.WordPtr(f.LAST - 2)
	t.Log("before", dumpmem(f, v, 10))
	v = f.WordPtr(v - 2)
	vl := f.WordPtr(v)
	t.Log("v&mask == ", v&0x0f0, vl)
	if vl&0x00f0 != IMEDD {
		t.Log("Here is LAST", dumpmem(f, v, 10))
		t.Log("couldn't find bit")
		t.Fail()
	}
}
func TestWordFromASMLabels(t *testing.T) {
	f := New(nil, nil)
	asm := `
;   ?DUP	( w -- w w | 0 )
;		Dup tos if its is not zero.

		$COLON	4,'TESTQDUP',QDUP
		DW	DUPP
		DW	QBRAN,QDUP1
		DW	DUPP
QDUP1:		DW	EXIT
`
	f.WordFromASM(asm)
	f.Push(0)
	RunWord("TESTQDUP", f, t)
	a := f.Pop()
	if a != 0 {
		t.Fatal(asm, "didn't leave 4 on the stack left", a)
	}
}

// Test that adding HiForth adds all the words
func TestAddHiForth(t *testing.T) {
	f := New(nil, nil)
	wordlist := []string{"doVAR", "UP", "doUSER", "SP0", "RP0", "'?KEY", "'EMIT", "'EXPECT", "'TAP", "'ECHO", "'PROMPT", "BASE", "tmp", "SPAN", ">IN", "#TIB", "CSP", "'EVAL", "'NUMBER", "HLD", "HANDLER", "CONTEXT", "CURRENT", "CP", "NP", "LAST", "doVOC", "FORTH", "?DUP", "ROT", "2DROP", "2DUP", "+", "D+", "NOT", "NEGATE", "DNEGATE", "-", "ABS", "=", "U<", "<", "MAX", "MIN", "WITHIN", "UM/MOD", "M/MOD", "/MOD", "MOD", "/", "UM*", "*", "M*", "*/MOD", "*/", "CELL+", "CELL-", "CELLS", "ALIGNED", "BL", ">CHAR", "DEPTH", "PICK", "+!", "2!", "2@", "COUNT", "HERE", "PAD", "TIB", "@EXECUTE", "CMOVE", "FILL", "-TRAILING", "PACK$", "DIGIT", "EXTRACT", "<#", "HOLD", "#", "#S", "SIGN", "#>", "str", "HEX", "DECIMAL", "DIGIT?", "NUMBER?", "?KEY", "KEY", "EMIT", "NUF?", "PACE", "SPACE", "SPACES", "TYPE", "CR", "do$", `$"|`, `."|`, ".R", "U.R", "U.", ".", "?", "parse", "PARSE", ".(", "(", `\`, "CHAR", "TOKEN", "WORD", "NAME>", "SAME?", "find", "NAME?", "^H", "TAP", "kTAP", "accept", "EXPECT", "QUERY", "CATCH", "THROW", "NULL$", "ABORT", `abort"`, "$INTERPRET", "[", ".OK", "?STACK", "EVAL", "PRESET", "xio", "FILE", "HAND", "I/O", "CONSOLE", "QUIT", "'", "ALLOT", ",", "[COMPILE]", "COMPILE", "LITERAL", `$,"`, "RECURSE", "FOR", "BEGIN", "NEXT", "UNTIL", "AGAIN", "IF", "AHEAD", "REPEAT", "THEN", "AFT", "ELSE", "WHILE", `ABORT"`, `$"`, `."`, "?UNIQUE", "$,n", "$COMPILE", "OVERT", ";", "]", "call,", ":", "IMMEDIATE", "USER", "CREATE", "VARIABLE", "_TYPE", "dm+", "DUMP", ".S", "!CSP", "?CSP", ">NAME", ".ID", "SEE", "WORDS", "VER", "hi", "'BOOT", "COLD"}
	for _, word := range wordlist {
		_, e := f.Addr(word)
		if e != nil {
			t.Log(e)
			t.Fail()
		}
	}
}

// CMOVE moves c bytes from a to b
func TestCmove(t *testing.T) {
	f := New(nil, nil)
	f.SetWordPtr(0x1a, 0x8)
	f.SetWordPtr(0x3b, 0x9)
	f.Push(0x1a)
	f.Push(0x3b)
	f.Push(1)
	RunWord("CMOVE", f, t)
	b := f.WordPtr(0x3b)
	if b != 0x8 {
		t.Fatal("b ought to be 8 but it was", b)
	}
}

func TestStackPtrs(t *testing.T) {
	f := New(nil, nil)
	RunWord("SP0", f, t)
	if a := f.WordPtr(f.Pop()); a != SPP {
		t.Log("WordPtr(SP0) is ", f.WordPtr(a))
		t.Log("SP0 ought to be SPP but it is", a, "not", SPP)
		t.Fail()
	}
	RunWord("RP0", f, t)
	if a := f.WordPtr(f.Pop()); a != RPP {
		t.Log("WordPtr(RP0) is ", f.WordPtr(a))
		t.Log("RP0 ought to be RPP but it is", a, "not", RPP)
		t.Fail()
	}
}

func TestInlineStrings(t *testing.T) {
	f := New(nil, nil)
	f.AddPrim("!IO", func() {
		f._next()
	}, 0)
	//RunWord("CR", f, t)
	addr, _ := f.Addr("hi")
	v := f.Memory[addr+10]
	t.Log(dumpmem(f, addr, 20))
	if v != 8 {
		t.Fatal("should be 8 was", v)
	}
	v = f.Memory[addr+11]
	if v != 'e' {
		t.Fatal("should be ", 'e', "but was ", v)
	}
}

func see(word string, f *Forth, t *testing.T) {
	if a, e := f.Addr(word); e != nil {
		t.Fail()
		t.Log(e, "couldn't find", a)
	} else {
		t.Log("see:", word, dumpmem(f, a, 20))
	}
}

func NewForth(in string) (o *bytes.Buffer, f *Forth) {
	o = new(bytes.Buffer)
	i := strings.NewReader(in)
	f = New(i, o)
	return
}

func TestDoTqp(t *testing.T) {
	b := new(bytes.Buffer)
	f := New(strings.NewReader("BYE"), b)
	good := `'gooood'`
	AddWord(f, t, "testme", "!IO", "DOTQP", good)
	RunWord("testme", f, t)
	res := string(b.Bytes())
	if res != good[1:len(good)-1] {
		t.Fatal("res should have been", good, "but was", res)
	}

}

func AddWord(f *Forth, t *testing.T, name string, words ...string) {
	a := append([]string{"CALLL", "doLIST"}, words...)
	a = append(a, "EXIT")
	err := f.compileWords(name, a, nil, 0)
	if err != nil {
		t.Fatal("ERROR compileWords:", err)
	}
}

func TestSubb(t *testing.T) {
	f := New(nil, nil)
	f.Push(10)
	f.Push(9)
	RunWord("-", f, t)
	z := f.Pop()
	if z != 1 {
		t.Fatal("Subb should have returned 1 but we got", z)
	}
}

func TestNpZero(t *testing.T) {
	f := New(nil, nil)
	AddWord(f, t, "NPis", "NP", "@")
	RunWord("NPis", f, t)
	z := f.Pop()
	if z != f.NP {
		t.Log("UPP memory is", dumpmem(f, UPP, 100), "UPP is", UPP)
		t.Log("UPP+35*CELLL memory is", dumpmem(f, UPP+35*CELLL, 10))
		t.Fatal("should have", f.NP, "but we have", z, "instead")
	}
}

func TestMain(t *testing.T) {
	i := strings.NewReader("BYE \r\n")
	o := new(bytes.Buffer)
	f := New(i, o)
	f.Main()
	t.Log(o)
	t.Log(i)
}

func TestNameq(t *testing.T) {
	f := New(nil, nil)
	AddWord(f, t, "test", "doLIT", "100", "SPAN", "!", "SPAN", "NAME?")
	RunWord("test", f, t)
	a := uint16(0x3e1c)
	i := f.Pop()
	j := f.Pop()
	if i != 0 {
		t.Fatal("should have been 0 but was", i)
	}
	if j != a {
		t.Fatal("should be", a, "but was", j)
	}
}

func TestDump(t *testing.T) {
	o := new(bytes.Buffer)
	f := New(os.Stdin, o)
	f.Push(0x35c4)
	f.Push(2400)
	RunWord("DUMP", f, t)
	t.Log(string(o.Bytes()))
}

func TestDotid(t *testing.T) {
	o := new(bytes.Buffer)
	i := os.Stdin
	f := New(i, o)
	f.Push(0x35c8)
	RunWord(".ID", f, t)
	z := string(o.Bytes())
	if z != "COLD" {
		t.Fatal("got ", z, "instead of COLD")
	}

}

func TestNotfound(t *testing.T) {
	i := strings.NewReader("NOTFOUND\r BYE\r")
	o := new(bytes.Buffer)
	f := New(i, o)
	f.Main()
	t.Log(string(o.Bytes()))
}


func TestWords(t *testing.T) {
	i := strings.NewReader("WORDS BYE\r")
	o := new(bytes.Buffer)
	f := New(i, o)
	f.Main()
	t.Log(string(o.Bytes()))
}

func TestNum(t *testing.T) {
	i := strings.NewReader("10 BYE\r")
	o := new(bytes.Buffer)
	f := New(i, o)
	f.Main()
	t.Log(string(o.Bytes()))
	z := f.Pop()
	if z != 10 {
		t.Fatal("forth should've left 10 on the stack but left", z)
	}

}

func TestColon(t *testing.T) {
	i := strings.NewReader(": boo 10 BYE ; NP @ 100 DUMP CP @ 100 - 100 DUMP boo\r")
	o := new(bytes.Buffer)
	f := New(i, o)
	f.Main()
	t.Log(string(o.Bytes()))
	z := f.Pop()
	if z != 10 {
		t.Fatal("forth should've left 10 on the stack but left", z)
	}

}
