package eforth

import (
	"fmt"
	"strconv"
	"strings"
)

var asm2forth map[string]string

func (f *Forth) compileWords(name string, words []string, labels map[string]uint16) (err error) {
	err = nil
	startaddr := CODEE + (2 * f.prims)
	setit := func(i int, addr uint16) {
		f.SetWordPtr(startaddr+uint16(i*CELLL), addr)
	}
	for i, word := range words {
		addr, e := f.Addr(word)
		b, ok := asm2forth[word]
		a, se := strconv.Atoi(word)
		c, e2 := f.Addr(b)
		v, el := labels[word]
		switch {
		case e == nil:
			setit(i, addr)
		case se == nil:
			addr = uint16(a)
			setit(i, addr)
		case el:
			addr = v * CELLL
			setit(i, addr+startaddr)
		case e2 == nil && ok:
			setit(i, c)
		default:
			err = e
			fmt.Println("ERROR: could not add", name, "because", e)
			return

		}
	}
	f.prim2addr[name] = startaddr
	f.AddName(name, startaddr)
	f.prims = f.prims + 2*uint16(len(words))
	return
}

func (f *Forth) WordFromASM(asm string) (err error) {
	words := []string{}
	labels := make(map[string]uint16)
	name := ""
	err = nil
	restart := func() {
		words = []string{}
		labels = make(map[string]uint16)
	}
	
	for _, line := range strings.Split(asm, "\n") {
		fields := strings.FieldsFunc(line, func(r rune) bool {
			if r == ' ' || r == ',' || r == '\t' {
				return true
			}
			return false
		})
		toks := fields
	tokenloop:
		for len(toks) > 0 {
			tok := toks[0]
			toks = toks[1:]
			switch {
			case tok == "$COLON":
				if name != "" {
					if err = f.compileWords(name, words, labels); err != nil {
						return err
					}
					restart()
				}
				name = toks[1]
				name = name[1 : len(name)-1]
				vname := fields[3]
				asm2forth[vname] = name
				words = append(words, []string{"2", ":"}...)
				break tokenloop
			case tok == "$USER":
				if name != "" {
					if err = f.compileWords(name, words, labels); err != nil {
						return err
					}
					restart()
				}
				name = toks[1]
				name = name[1 : len(name)-1]
				vname := fields[3]
				asm2forth[vname] = name
				words = append(words, []string{"2", ":", "doUSER", strconv.Itoa(int(f._USER))}...)
				f._USER += CELLL
				break tokenloop
			case strings.HasSuffix(tok, ":") && tok == fields[0]:
				label := tok
				label = label[:len(label)-1]
				labels[label] = uint16(len(words))
			case tok == "DW":
				words = append(words, toks...)
				break tokenloop
			}
		}
	}
	err = f.compileWords(name, words, labels)
	return err

}

func (f *Forth) AddHiforth() {
	f.prim2addr["UPP"] = UPP
	asm2forth = make(map[string]string)
	f.macros = make(map[string]fn)
	f.macros["#TIB"] = func() {
		f._USER = f._USER + CELLL
	}
	amap := []struct {
		aword string
		fword string
	}{
		{"DOLIT", "doLIT"},
		{"QBRAN", "?BRANCH"},
		{"DUPP", "DUP"},
		{"RFROM", "R>"},
		{"UPLUS", "UM+"},
		{"TOR", ">R"},
		{"AT", "@"},
	}
	for _, v := range amap {
		asm2forth[v.aword] = v.fword
	}
	hiforth := []string{
`

;; System and user variables

;   doVAR	( -- a )
;		Run time routine for VARIABLE and CREATE.

		$COLON	COMPO+5,'doVAR',DOVAR
		DW	RFROM,EXIT

;   UP		( -- a )
;		Pointer to the user area.

		$COLON	2,'UP',UP
		DW	DOVAR
		DW	UPP

;   ROT		( w1 w2 w3 -- w2 w3 w1 )
;		Rot 3rd item to top.

		$COLON	3,'ROT',ROT
		DW	TOR,SWAP,RFROM,SWAP,EXIT


;   +		( w w -- sum )
;		Add top two items.

		$COLON	1,'+',PLUS
		DW	UPLUS,DROP,EXIT


;   doUSER	( -- a )
;		Run time routine for user variables.

		$COLON	COMPO+6,'doUSER',DOUSE
		DW	RFROM,AT,UP,AT,PLUS,EXIT

;   SP0		( -- a )
;		Pointer to bottom of the data stack.

		$USER	3,'SP0',SZERO

;   RP0		( -- a )
;		Pointer to bottom of the return stack.

		$USER	3,'RP0',RZERO

;   '?KEY	( -- a )
;		Execution vector of ?KEY.

		$USER	5,"'?KEY",TQKEY

;   'EMIT	( -- a )
;		Execution vector of EMIT.

		$USER	5,"'EMIT",TEMIT

;   'EXPECT	( -- a )
;		Execution vector of EXPECT.

		$USER	7,"'EXPECT",TEXPE

;   'TAP	( -- a )
;		Execution vector of TAP.

		$USER	4,"'TAP",TTAP

;   'ECHO	( -- a )
;		Execution vector of ECHO.

		$USER	5,"'ECHO",TECHO

;   'PROMPT	( -- a )
;		Execution vector of PROMPT.

		$USER	7,"'PROMPT",TPROM

;   BASE	( -- a )
;		Storage of the radix base for numeric I/O.

		$USER	4,'BASE',BASE

;   tmp		( -- a )
;		A temporary storage location used in parse and find.

		$USER	COMPO+3,'tmp',TEMP

;   SPAN	( -- a )
;		Hold character count received by EXPECT.

		$USER	4,'SPAN',SPAN

;   >IN		( -- a )
;		Hold the character pointer while parsing input stream.

		$USER	3,'>IN',INN

;   #TIB	( -- a )
;		Hold the current count and address of the terminal input buffer.

		$USER	4,'#TIB',NTIB
		_USER = _USER+CELLL
`,
	}

	for _, asm := range hiforth {
		err := f.WordFromASM(asm)
		if err != nil {
			fmt.Println("ERROR: ", err)
		}
	}

}
