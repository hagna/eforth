package eforth

import (
	"bytes"
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
	f.AddPrim("NEXT", f._next)
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
	for _, v := range []string{"CALL", ":", ";"} {
		aa, _ := f.Addr(v)
		good = append(good, aa)
	}
	good[0] = 2
	start, _ := f.Addr("nop")
	cmp := []uint16{f.WordPtr(start), f.WordPtr(start + 2), f.WordPtr(start + 4)}
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
	for _, v := range []string{"CALL", ":", "nop", ";"} {
		aa, _ := f.Addr(v)
		good = append(good, aa)
	}
	good[0] = 2
	start, _ := f.Addr("bar")
	cmp := []uint16{f.WordPtr(start), f.WordPtr(start + 2), f.WordPtr(start + 4), f.WordPtr(start + 6)}
	if !Uint16sEqual(cmp, good) {
		t.Fatal("code of nop wasn't", good, "but was", cmp)
	}
	anop, _ := f.Addr("nop")
	abar, _ := f.Addr("bar")
	if abar-anop != 6 {
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
	for _, v := range []string{"CALL", ":", "nop", ";"} {
		aa, _ := f.Addr(v)
		good = append(good, aa)
	}
	good[0] = 2
	start, _ := f.Addr("bar")
	cmp := []uint16{f.WordPtr(start), f.WordPtr(start + 2), f.WordPtr(start + 4), f.WordPtr(start + 6)}
	if !Uint16sEqual(cmp, good) {
		t.Fatal("code of nop wasn't", good, "but was", cmp)
	}
}

// ignore comments for now
func TestAddWordComments(t *testing.T) {
	f := NewForth()
	f.AddWord(": nop ( -- ) ;")
	good := []uint16{}
	for _, v := range []string{"CALL", ":", ";"} {
		aa, _ := f.Addr(v)
		good = append(good, aa)
	}
	good[0] = 2
	start, _ := f.Addr("nop")
	cmp := []uint16{f.WordPtr(start), f.WordPtr(start + 2), f.WordPtr(start + 4)}
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

/*
	Test the inner interpreter as they call it.  It's supposed to use NEXT doLIST
	and EXIT to loop through the high level forth till it finds primitives or
	machine code code words instead of colon words to execute.
*/
func TestMain(t *testing.T) {
	f := NewForth()
	called := false
	f.AddPrim("BYE", func() {
		fmt.Println("hello from the BYE primitive")
		called = true
		f.BYE()
	})
	f.AddWord(": nop ;")
	f.AddWord(": C BYE ;")
	f.AddWord(": B C ;")
	f.AddWord(": A nop B ;")
	a, err := f.Addr("A")
	fmt.Printf("Addr of doit is %x\n", a)
	if err != nil {
		t.Fatal(err)
	}
	f.SetWordPtr(COLDD, a)
	f.IP = COLDD
	f._next()
	f.Main()
	if !called {
		t.Fatal("Didn't call BYE function")
	}
}

/*
   Name dictionary entries or records or headers in order of my favorite descriptions
   contain:
       addr of code word
       link to previous name
       len prefixed string
   They start on CELLL boundaries 16 bits or 2 bytes as of this writing
*/
func TestNamedict(t *testing.T) {
	f := NewForth()
	s := f.Memory[NAMEE-8 : NAMEE]
	fmt.Println("range is ", NAMEE-8, "to", NAMEE)
	good := []byte{0x80, 0x1, 0, 0, 3, 0x42, 0x59, 0x45} // ASSUME BYE is the first primitive
	if !bytes.Equal(s, good) {
		t.Fatal("Wrong memory was", s, "and it should be", good)
	}
}

func TestNamedDictColons(t *testing.T) {
    f := NewForth()
    oldNP := f.NP
    f.AddWord(": nop ;")
    newNP := f.NP
    if oldNP == newNP {
        t.Fatal("adding a word should change the NP pointer, but it didn't", oldNP)
    }
}

func RunWord(word string, f *Forth, t *testing.T) {
	called := false
	f.AddPrim("BYE", func() {
		fmt.Println("hello from the BYE primitive")
		called = true
		f.BYE()
	})
	f.AddWord(fmt.Sprintf(": A %s BYE ;", word))
	a, err := f.Addr("A")
	fmt.Printf("Addr of doit is %x\n", a)
	if err != nil {
		t.Fatal(err)
	}
	f.SetWordPtr(COLDD, a)
	f.IP = COLDD
	f._next()
	f.Main()
	if !called {
		t.Fatal("Didn't call BYE function")
	}
}

// IF THEN ought to compile correctly when AddWord is called
func TestIfThen(t *testing.T) {
	f := NewForth()
	f.AddWord(": ?DUP ( w -- w w | 0 ) DUP IF DUP THEN ;")
	a, _ := f.Addr("?DUP")
	for _, j := range f.Memory[a:a+16] {
		fmt.Printf("%x ", j)
	}
	f.Push(30)
	RunWord("?DUP", f, t)
	x := f.Pop()
	y := f.Pop()
	fmt.Println()
	if x != 30 {
		t.Fatal("should have been 30 but was", x)
	}
	if x != y {
		t.Fatal("should have two equal vals on the stack but got", x, y)
	}
}

// nested IF THEN ought to compile correctly when AddWord is called
func TestIfThenNest(t *testing.T) {
	f := NewForth()
	f.AddWord(": nest ( w -- w w | 0 ) DUP IF DUP IF DUP + THEN DUP + THEN ;")
	a, _ := f.Addr("nest")
	for _, j := range f.Memory[a:a+16] {
		fmt.Printf("%x ", j)
	}
	f.Push(30)
	RunWord("nest", f, t)
	x := f.Pop()
	fmt.Println()
	if x != 120 {
		t.Fatal("should have been 120 but was", x)
	}
}   

func TestLifo(t *testing.T) {
	f := NewForth()
	f.Push(10)
	f.Push(20)
	s := f.Pop()
	if s != 20 {
		t.Fatal("fifo not lifo")
	}
}

func dumpmem(f *Forth, i, k uint16) string {
	res := ""
	for _, j := range f.Memory[i:i+k] {
		res += fmt.Sprintf("%x ", j)
	}
	res += "\n"
	return res
}

// IF THEN ELSE ought to compile correctly when AddWord is called
func TestIfThenElse(t *testing.T) {
	f := NewForth()
	f.AddWord(": ?DUP ( a w -- a | w ) DUP IF DROP ELSE SWAP DROP THEN ;")
	a, _ := f.Addr("?DUP")
	for _, j := range f.Memory[a:a+16] {
		fmt.Printf("%x ", j)
	}
	f.Push(30)
	f.Push(10)
	RunWord("?DUP", f, t)
	x := f.Pop()
	y := f.Pop()
	fmt.Println("x is ",x, "and y is", y)
	if x != 30 {
		t.Fatal("should have been 30 but was", x)
	}
	f.Push(30)
	f.Push(0)
	RunWord("?DUP", f, t)
	x = f.Pop()
	y = f.Pop()
	fmt.Println("x is ",x, "and y is", y)
	if x != 0 {
		t.Fatal("should have been 0 but was", x)
	}
}

// test begin again infinite loop
func TestBeginAgain(t *testing.T) {
	f := NewForth()
	word :=  ": forever BEGIN DUP AGAIN ;"
	f.AddWord(word)
	f.Push(99)
	a, _ := f.Addr("forever")
	i := a+4*CELLL
	branchto := f.WordPtr(i)
	waddr := f.WordPtr(branchto)
	pcode := f.WordPtr(waddr)
	word, ok := f.pcode2word[pcode]
	if !ok {
		t.Log("The word", word, "in bytes is", dumpmem(f, a, 6*CELLL))
		t.Log("Found branch to address", branchto, "which is ", waddr)
		t.Fatal("was expecting to find pcode for", pcode)
	}
	if word != "DUP" {
		t.Log("The word", word, "in bytes is", dumpmem(f, a, 6*CELLL))
		t.Log("pointer deref after again was", waddr)
		t.Log("deref that and we ought to have pcode for DUP", pcode)
		t.Fatal("should have found DUP by following that address")
	}
}

// begin until loops
func TestBeginUntil(t *testing.T) {
	f := NewForth()
	word := ": oneiter BEGIN doLIT 1 UNTIL doLIT 2 ;"
	f.AddWord(word)
	RunWord("oneiter", f, t)
	a := f.Pop()
	if a != 2  {
		t.Fatal(word, "should have left 2 on the stack")
	}
}

func TestBeginUntil10(t *testing.T) {
	f := NewForth()
	called := 0
	f.AddPrim("CALLME", func() {
		called += 1
		f._next()
	})
	word := ": teniter doLIT 10 BEGIN CALLME doLIT -1 + DUP doLIT 0 = UNTIL doLIT 2 ;"
	f.AddWord(word)
	RunWord("teniter", f, t)
	if called != 10  {
		t.Fatal(word, "called CALLME", called, "times instead of 10")
	}
}
	
