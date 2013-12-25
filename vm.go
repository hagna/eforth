package eforth

// following tutorial at http://www.offete.com/files/zeneForth.htm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"strconv"
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

	LAST uint16 // last name in name dictionary
	NP   uint16 // bottom of name dictionary
}

type fn func()

func wordptr(mem []byte, reg uint16) (res uint16) {
	res = binary.LittleEndian.Uint16(mem[reg:])
	return
}

func setwordptr(mem []byte, reg, value uint16) {
	binary.LittleEndian.PutUint16(mem[reg:], value)
}

func NewForth() *Forth {
	f := &Forth{SP: SPP, RP: RPP,
		prim2addr:  make(map[string]uint16),
		prim2func:  make(map[string]fn),
		pcode2word: make(map[uint16]string),
		NP:         NAMEE,
		LAST:       0}
	fmt.Printf("NAMEE is %x\n", NAMEE)
	words := []struct {
		word string
		m    fn
	}{
		{"BYE", f.BYE},
		{"CALL", f.Call},
		{":", f.doLIST},
		{"!IO", f.B_IO},
		{"?RX", f.Q_RX},
		{"!TX", f.B_TX},
		{"EXECUTE", f.Execute},
		{"doLIT", f.doLIT},
		{";", f.EXIT},
		{"EXIT", f.EXIT},
		{"NEXT", f.Next},
		{"?BRANCH", f.Q_branch},
		{"BRANCH", f.Branch},
		{"!", f.Bang},
		{"@", f.At},
		{"C!", f.Cbang},
		{"RP@", f.RPat},
		{"RP!", f.RPbang},
		{"R>", f.Rfrom},
		{"R@", f.Rat},
		{">R", f.Tor},
		{"DROP", f.Drop},
		{"DUP", f.Dup},
		{"SWAP", f.Swap},
		{"OVER", f.Over},
		{"SP@", f.Sp_at},
		{"SP!", f.Sp_bang},
		{"0<", f.Zless},
		{"AND", f.And},
		{"OR", f.Or},
		{"XOR", f.Xor},
		{"UM+", f.UMplus},
	}
	f.prim2addr["UPP"] = UPP
	for _, v := range words {
		f.AddPrim(v.word, v.m)
	}
	f.ColonDefs()
	return f
}

func (f *Forth) AddName(word string, addr uint16) {
	fmt.Println("AddName(", word, ", ", addr, ")")
	_len := uint16(len(word) / CELLL)  // rounded down cell count
	f.NP = f.NP - ((_len + 3) * CELLL) // new header on cell boundary
	i := f.NP
	fmt.Printf("writing to memory address %x\n", i)
	f.SetWordPtr(i, addr)
	f.SetWordPtr(i+2, f.LAST)
	f.LAST = uint16(i + 4)
	f.Memory[f.LAST] = byte(len(word))
	for j, c := range word {
		f.Memory[int(i)+5+j] = byte(c)
	}

}

func (f *Forth) AddPrim(word string, m fn) {
	f.prims = f.prims + 1
	addr := CODEE + (2 * (f.prims - 1))
	f.prim2addr[word] = addr
	f.prim2func[word] = m
	f.pcode2word[f.prims] = word
	fmt.Printf("%x is \"%s\"\n", f.prims, word)
	f.SetWordPtr(addr, f.prims)
	f.AddName(word, addr)
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
	li := len(ifs) -1 
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
	addr := CODEE + (2 * (prims - 1))
	name := all[1]
	all[1] = ":"
	f.prim2addr[name] = addr
	f.AddName(name, addr)
	iwords := all[1:]
	f.SetWordPtr(addr, 2) // CALL is 2
	ifs := []uint16{}
	begins := []uint16{}
	for j, word := range iwords {
		fmt.Println("word is ", word)
		switch(word) {
			case "BEGIN":
				begins = append(begins, addr+2)
				// get rid of +2 makes no sense
			case "AGAIN":
				i := len(begins) -1
				beginaddr := begins[i]
				begins = begins[:i]
				branch, _ := f.Addr("BRANCH")
				f.SetWordPtr(addr+2, branch)
				f.SetWordPtr(addr+4, beginaddr)
				prims = prims + 2
				addr = addr + 4
			case "IF":
				doIF(f, addr, &ifs, "?BRANCH")
				prims = prims + 2
				addr = addr + 4
			case "THEN":
				/*
				CALL addr addr IF addr addr THEN addrA
				CALL addr addr QBRAN p_addrA addr addr addrA

				Also things could be nested
				*/
				doTHEN(f, ifs, addr+2)
				// +2 is because addr points to the previous word addr
			case "ELSE":
				/*
				CALL addr addr IF addr ELSE addrA addr THEN addrB 
				CALL addr addr QBRAN p_addrA addr BRAN p_addrB addrA addr addrB
				*/
				doTHEN(f, ifs, addr+6)
				doIF(f, addr, &ifs, "BRANCH")
				prims = prims + 2
				addr = addr + 4
			default:
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
						return
					}
					wa = uint16(x)
			
				}
				addr = addr + 2
				f.SetWordPtr(addr, wa)
				prims = prims + 1
				fmt.Printf("%x: %x %s\n", addr, wa, word)
		}
		fmt.Printf("addr is %x\n", addr)
	}

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

func (f *Forth) CallFn(word string) error {
	m, ok := f.prim2func[word]
	fmt.Printf("CallFn %v \"%s\"\n", m, word)
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

// this simulates the von neuman machine or processor
func (f *Forth) Main() {
	f.B_IO()
	fmt.Println("---------Main----------")
	var pcode uint16
	var word string
inf:
	for {
		// simulate JMP to f.WP
		pcode = f.WordPtr(f.WP)
		word = f.Frompcode(pcode)
		fmt.Printf("WP %x IP %x pcode %x word \"%s\"\n", f.WP, f.IP, pcode, word)
		err := f.CallFn(word)
		if err != nil {
			fmt.Println(err)
			break inf
		}
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

/*
lodsw
jmp ax
*/
func (f *Forth) _next() {
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
