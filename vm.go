package eforth

// following tutorial at http://www.offete.com/files/zeneForth.htm

import (
	"encoding/binary"
	"fmt"
)

/*

Memory used in eForth is separated into the following areas:

Cold boot         100H-17FH         Cold start and variable initial values
Code dictionary   180H-1344H        Code dictionary growing upward
Free space        1346H-33E4H       Shared by code and name dictionaries
Name/word         33E6H-3BFFH       Name dictionary growing downward
Data stack        3C00H-3E7FH       Growing downward
TIB               3E80H-            Growing upward
Return stack      -3F7FH            Growing downward
User variables    3F80H-3FFFH

*/

const (
	CELLL = 2              // size of cell
	EM    = 0x04000        // top of memory
	COLDD = 0x00100        // cold start vector
	US    = 64 * CELLL     // user area size in cells
	RTS   = 64 * CELLL     // return stack/TIB size
	RPP   = EM - 8*CELLL   // start of return stack (RP0)
	TIBB  = RPP - RTS      // terminal input buffer (TIB)
	SPP   = TIBB - 8*CELLL // start of data stack (SP0)
	UPP   = EM - 256*CELLL // start of user area (UP0)
	NAMEE = UPP - 8*CELLL  //name dictionary
	CODEE = COLDD + US     // code dictionary
)

// 76 13 e8 35 3 62 61 72 == *bar where bar is the last field and the others are the code reference and prev word reference
type Forth struct {
	/*

	   Forth Register 8086 Register               Function

	   IP  SI                         Interpreter Pointer
	   SP  SP                         Data Stack Pointer
	   RP  BP                         Return Stack Pointer
	   WP  AX                         Word or Work Pointer
	   UP  (in memory )               User Area Pointer

	*/

	IP uint16
	SP uint16
	RP uint16
	WP uint16

	input  interface{}
	output interface{}

	Memory [EM]byte

	/*
	   primitive words or code words are as follows:

	   System interface:       BYE, ?rx, tx!, !io
	   Inner interpreters:     doLIT, doLIST, next, ?branch,  branch, EXECUTE, EXIT
	   Memory access:          ! , @,  C!,  C@
	   Return stack:           RP@,  RP!,  R>, R@,  R>
	   Data stack:             SP@,  SP!,  DROP, DUP,  SWAP,  OVER
	   Logic:                  0<,  AND,  OR,  XOR
	   Arithmetic:             UM+

	*/

	/*
	   For setting up the primitive words in memory
	   and interpretting pcode.
	*/

	prims      uint16
	prim2addr  map[string]uint16
	prim2func  map[string]fn
	pcode2word map[uint16]string
}

type fn func()

func wordptr(mem []byte, reg uint16) (res uint16) {
	res = binary.BigEndian.Uint16(mem[reg:])
	return
}

func setwordptr(mem []byte, reg, value uint16) {
	binary.BigEndian.PutUint16(mem[reg:], value)
}

func NewForth() *Forth {
	f := &Forth{SP: SPP, RP: RPP,
		prim2addr:  make(map[string]uint16),
		prim2func:  make(map[string]fn),
		pcode2word: make(map[uint16]string)}
	words := []struct {
		word string
		m    fn
	}{
		{":", f.doLIST},
		{"!io", f.B_IO},
		{"bye", f.BYE},
		{"?rx", f.Q_RX},
		{"!tx", f.B_TX},
		{"execute", f.Execute},
		{"doLIT", f.doLIT},
		{";", f.EXIT},
		{"bye", f.BYE},
		{"?rx", f.Q_RX},
		{"!tx", f.B_TX},
		{"next", f.Next},
		{"?branch", f.Q_branch},
		{"branch", f.Branch},
		{"!", f.Bang},
		{"@", f.At},
		{"c!", f.Cbang},
		{"rp@", f.RPat},
		{"rp!", f.RPbang},
		{"r>", f.Rfrom},
		{"r@", f.Rat},
		{">r", f.Tor},
		{"drop", f.Drop},
		{"dup", f.Dup},
		{"swap", f.Swap},
		{"over", f.Over},
		{"sp@", f.Sp_at},
		{"sp!", f.Sp_bang},
		{"0<", f.Zless},
		{"and", f.And},
		{"or", f.Or},
		{"xor", f.Xor},
		{"um+", f.UMplus},
	}
	for _, v := range words {
		f.AddPrim(v.word, v.m)
	}
	fmt.Println("NewForth()")
	return f
}

func (f *Forth) AddPrim(word string, m fn) {
	f.prims = f.prims + 1
	addr := CODEE + (2 * (f.prims - 1))
	f.prim2addr[word] = addr
	f.prim2func[word] = m
	f.pcode2word[f.prims] = word
	f.SetWordPtr(addr, f.prims)
}

func (f *Forth) Addr(word string) uint16 {
	return f.prim2addr[word]
}

func (f *Forth) Call(word string) {
	m := f.prim2func[word]
	fmt.Println("m is", m)
	m()
}

func (f *Forth) Frompcode(pcode uint16) (res string) {
	res = f.pcode2word[pcode]
	return
}

// this simulates the von neuman machine or processor
func (f *Forth) Main() {
	f.B_IO()
	var pcode uint16
	var word string
inf:
	for {
		pcode = f.WP
		word = f.Frompcode(pcode)
		f.Call(word)
		if f.IP == 0xffff { // for BYE
			break inf
		}
	}
}

func (f *Forth) WordPtr(reg uint16) (res uint16) {
	return wordptr(f.Memory[0:], reg)
}

func (f *Forth) SetWordPtr(reg, value uint16) {
	setwordptr(f.Memory[0:], reg, value)
}

func (f *Forth) RegLower(w uint16) (res byte) {
	res = byte(0x00ff & w)
	return
}

func (f *Forth) SetBytePtr(i uint16, v byte) {
	f.Memory[i] = v
}

// NEXT ought to be a macro it sets WP to the next instruction
// and increments the instruction pointer
func (f *Forth) NEXT() {
	fmt.Println("IP is", f.IP)
	f.WP = f.WordPtr(f.IP) // same as LODSW puts [IP] into WP and advances IP
	f.IP += 2
}

// swap register values
func XCHG(a, b *uint16) {
	olda := *a
	*a = *b
	*b = olda
}

// PUSH is
// SP = SP -2
// [SP] = operand
func (f *Forth) Push(v uint16) {
	fmt.Println("f.SP is", f.SP)
	f.SP = f.SP - 2
	binary.BigEndian.PutUint16(f.Memory[f.SP:], v)
}

// POP is
// operand = [SP]
// SP = SP + 2
func (f *Forth) Pop() uint16 {
	res := binary.BigEndian.Uint16(f.Memory[f.SP:])
	f.SP = f.SP + 2
	return res
}
