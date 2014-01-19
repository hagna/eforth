package eforth

// following tutorial at http://www.offete.com/files/zeneForth.htm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
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
	BASEE   = 10 // default radix
	CELLL   = 2  // size of cell
	VOCSS   = 8
	EM      = 0x04000        // top of memory
	COLDD   = 0x00100        // cold start vector
	US      = 64 * CELLL     // user area size in cells
	RTS     = 64 * CELLL     // return stack/TIB size
	RPP     = EM - 8*CELLL   // start of return stack (RP0)
	TIBB    = RPP - RTS      // terminal input buffer (TIB)
	SPP     = TIBB - 8*CELLL // start of data stack (SP0)
	UPP     = EM - 256*CELLL // start of user area (UP0)
	NAMEE   = UPP - 8*CELLL  //name dictionary
	CODEE   = COLDD + US     // code dictionary
	CALLL   = 2
	VERSION = 1
	COMPO   = 0x040
	IMEDD   = 0x080
	MASKK   = 0x07F1F
)

// this change could make it easier to port to new word sizes
// but for now we'll use this
func asint16(u uint16) int16 {
	b := new(bytes.Buffer)
	if err := binary.Write(b, binary.LittleEndian, &u); err != nil {
		fmt.Println(err)
	}
	var res int16
	if err := binary.Read(b, binary.LittleEndian, &res); err != nil {
		fmt.Println(err)
	}
	return res
}

func asuint16(i int16) uint16 {
	b := new(bytes.Buffer)
	if err := binary.Write(b, binary.LittleEndian, &i); err != nil {
		fmt.Println(err)
	}
	var res uint16
	if err := binary.Read(b, binary.LittleEndian, &res); err != nil {
		fmt.Println(err)
	}
	return res
}

type word [CELLL]byte // the word size

func (v *word) uset(n uint) {
	z := uint16(n)
	b := new(bytes.Buffer)
	if err := binary.Write(b, binary.LittleEndian, &z); err != nil {
		fmt.Println(err)
	}
	for i := 0; i < CELLL; i++ {
		v[i] = b.Bytes()[i]
	}
}

func (v *word) set(n int) {
	z := int16(n)
	b := new(bytes.Buffer)
	if err := binary.Write(b, binary.LittleEndian, &z); err != nil {
		fmt.Println(err)
	}
	for i := 0; i < CELLL; i++ {
		v[i] = b.Bytes()[i]
	}
}

func (v *word) signed() int {
	var res int16
	if err := binary.Read(bytes.NewBuffer(v[:]), binary.LittleEndian, &res); err != nil {
		fmt.Println(err)
	}
	return int(res)

}

func (v *word) unsigned() uint {
	var res uint16
	if err := binary.Read(bytes.NewBuffer(v[:]), binary.LittleEndian, &res); err != nil {
		fmt.Println(err)
	}
	return uint(res)

}

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

	IP  uint16
	SP  uint16
	RP  uint16
	WP  uint16
	aWP uint16

	Input  io.Reader
	Output io.Writer

	rxchan chan uint16

	Memory [EM]byte

	/*
	   primitive words or code words are as follows:

	   System interface:       _BYE, ?rx, tx!, !io
	   Inner interpreters:     doLIT, _doLIST, next, ?branch,  branch, EXECUTE, _EXIT
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

	prims      uint16 //used as definition counter or number of words
	prim2addr  map[string]uint16
	addr2word  map[uint16]string
	prim2func  map[string]fn
	pcode2word map[uint16]string

	LAST uint16 // last name in name dictionary
	NP   uint16 // bottom of name dictionary

	_USER  uint16        // first user variable offset
	macros map[string]fn // need this for hardcoding "_USER = ..." for #TIB, CONTEXT, and CURRENT user vars
}

func (f *Forth) NewWord(name string, startaddr uint16, bitmask int) {
	f.addr2word[startaddr] = name
	f.prim2addr[name] = startaddr
	f.AddName(name, startaddr, bitmask)
}

type fn func()

func wordptr(mem []byte, reg uint16) (res uint16) {
	res = binary.LittleEndian.Uint16(mem[reg:])
	return
}

func setwordptr(mem []byte, reg, value uint16) {
	binary.LittleEndian.PutUint16(mem[reg:], value)
}

func New(r io.Reader, w io.Writer) *Forth {
	f := &Forth{SP: SPP, RP: RPP,
		prim2addr:  make(map[string]uint16),
		addr2word:  make(map[uint16]string),
		prim2func:  make(map[string]fn),
		pcode2word: make(map[uint16]string),
		NP:         NAMEE,
		LAST:       0,
		_USER:      4 * CELLL,
		Input:      r,
		Output:     w,
	}
	f.AddPrimitives()
	f.AddHiforth()
	return f
}

func (f *Forth) AddName(word string, addr uint16, bitmask int) {
	//	fmt.Printf("AddName(%v, %x, %x\n", word, addr, bitmask)
	_len := uint16(len(word) / CELLL)  // rounded down cell count
	f.NP = f.NP - ((_len + 3) * CELLL) // new header on cell boundary
	i := f.NP
	f.SetWordPtr(i, addr)
	f.SetWordPtr(i+2, f.LAST)
	f.LAST = uint16(i + 4)
	l := byte(len(word))
	if bitmask != 0 {
		l = byte(bitmask) | l
	}
	f.Memory[f.LAST] = l
	for j, c := range word {
		f.Memory[int(i)+5+j] = byte(c)
	}

}

func (f *Forth) AddPrim(word string, m fn, flags int) {
	f.prims = f.prims + 1
	addr := CODEE + (2 * (f.prims - 1))
	f.prim2addr[word] = addr
	f.prim2func[word] = m
	f.pcode2word[f.prims] = word
	//fmt.Printf("%x is \"%s\"\n", f.prims, word)
	f.SetWordPtr(addr, f.prims)
	f.NewWord(word, addr, flags)
}

func (f *Forth) RemoveComments(a string) (b string) {
	b = a
	i := strings.Index(a, "(")
	if i == -1 {
		return
	} else {
		j := strings.Index(a, ")")
		b = a[:i] + a[j+1:]
	}
	return
}

func doTHEN(f *Forth, ifs []uint16, addr uint16) {
	li := len(ifs) - 1
	ifaddr := ifs[li]
	ifs = ifs[:li]
	f.SetWordPtr(ifaddr, addr)
}

func doIF(f *Forth, addr uint16, ifs *[]uint16, word string) {
	*ifs = append(*ifs, addr+4)
	w, _ := f.Addr(word)
	f.SetWordPtr(addr+2, w)
}

func (f *Forth) AddWord(cdef string) (e error) {
	prims := f.prims + 1
	e = nil
	all := strings.Fields(f.RemoveComments(cdef))
	startaddr := CODEE + (2 * (prims - 1))
	addr := startaddr
	name := all[1]
	all[1] = "_doLIST"
	iwords := all[1:]
	fmt.Println("AddWord:", name)
	for j, word := range iwords {
		fmt.Println("\t", word)
		var wa uint16
		if j > 1 && iwords[j-1] == "doLIT" {
			x, err := strconv.Atoi(word)
			if err != nil {
				e = err
				return
			}
			wa = uint16(x)
		} else {
			x, err := f.Addr(word)
			if err != nil {
				e = err
				fmt.Printf("ERROR: not adding \"%v\" because %v\n", name, err)
				return
			}
			wa = uint16(x)

		}
		addr = addr + 2
		f.SetWordPtr(addr, wa)
		prims = prims + 1
		fmt.Printf("%x: %x %s\n", addr, wa, word)
		fmt.Printf("addr is %x\n", addr)
	}

	f.SetWordPtr(startaddr, CALLL) // CALL is 2
	f.prim2addr[name] = startaddr
	f.AddName(name, startaddr, 0)
	f.prims = prims
	return
}

func (f *Forth) Addr(word string) (res uint16, err error) {
	err = nil
	res, ok := f.prim2addr[word]
	if !ok {
		err = errors.New(fmt.Sprintf(`Address for word "%s" not found`, word))
	}
	return
}

func (f *Forth) _CallFn(word string) error {
	m, ok := f.prim2func[word]
	//fmt.Printf("_CallFn %v \"%s\"\n", m, word)
	if !ok {
		return errors.New(fmt.Sprintf("No method found for \"%s\"", word))
	}
	m()
	return nil
}

func (f *Forth) Frompcode(pcode uint16) (res string) {
	res = f.pcode2word[pcode]
	return
}

func (f *Forth) setupIP() error {
	if IP, e := f.Addr("COLD"); e != nil {
		return e
	} else {
		f.SetWordPtr(COLDD, IP)
		f.IP = COLDD
	}
	f._next()
	return nil
}

func (f *Forth) showstacks() {
	fn := func(i, k uint16) string {
		var res string
		for j := k; j < i; j += 2 {
			a := binary.LittleEndian.Uint16(f.Memory[j:])
			res = fmt.Sprintf("%x ", a) + res
		}
		return res
	}
	a := "D: " + fn(SPP, f.SP)
	b := "R: " + fn(RPP, f.RP)
	fmt.Println(a + "\n" + b + "\n")
}

// this simulates the von neuman machine or processor
func (f *Forth) Main() {
	if e := f.setupIP(); e != nil {
		fmt.Println(e)
		return
	}
	var pcode uint16
	var word string
	debug := false
inf:
	for {

		if debug {
			fmt.Printf("&WP %x WP %x IP %x", f.aWP, f.WP, f.IP)
		}
		// simulate JMP to f.WP
		pcode = f.WordPtr(f.WP)
		word = f.Frompcode(pcode)
		if debug {
			calling, ok := f.addr2word[f.WP]
			s := word
			if ok {
				s = "::" + calling
			}
			fmt.Println("", s)
			f.showstacks()
			//fmt.Println(dumpmem(f, f.LAST-10, 20))
		}
		err := f._CallFn(word)
		if err != nil {
			fmt.Println(err)
			break inf
		}
		if f.IP == 0xffff { // for _BYE
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

/*
lodsw
jmp ax
*/
func (f *Forth) _next() {
	f.aWP = f.IP
	f.WP = f.WordPtr(f.IP)
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
	f.SP = f.SP - 2
	binary.LittleEndian.PutUint16(f.Memory[f.SP:], v)
}

// POP is
// operand = [SP]
// SP = SP + 2
func (f *Forth) Pop() uint16 {
	res := binary.LittleEndian.Uint16(f.Memory[f.SP:])
	f.SP = f.SP + 2
	return res
}
