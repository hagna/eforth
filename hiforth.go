package eforth

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var asm2forth map[string]string

type codeitem struct {
	val    []byte
	offset uint16
	name   string
	li     int
}

type codeList struct {
	lst         *[]codeitem
	startoffset uint16
}

func (c *codeList) add(name string, addr uint16) {
	slice := *c.lst

	offset := c.startoffset
	if len(slice) >= 1 {
		prev := slice[len(slice)-1]
		offset = prev.offset + uint16(len(prev.val))
	}
	var twobyte [CELLL]byte
	val := []byte{}
	val = append(val, twobyte[0:]...)
	binary.LittleEndian.PutUint16(val, addr)
	res := codeitem{val, offset, name, -1}
	*c.lst = append(*c.lst, res)
}

func (c *codeList) println() {
	slice := *c.lst
	for _, k := range slice {
		fmt.Printf("%x: ", k.offset)
		s := k.name
		if k.name == "STRING" {
			l := k.val[0]
			s = string(k.val[1:])[:l]
			s = fmt.Sprintf("%x%s", l, "'"+s+"'")
		}
		fmt.Printf("%s", s)
		if k.li != -1 {
			fmt.Printf(" [%x]", slice[k.li].offset)
		}
		fmt.Println()
	}
}

func (c *codeList) intoForth(f *Forth) {
	slice := *c.lst
	for _, v := range slice {
		vlst := v.val
		for i, b := range vlst {
			f.Memory[v.offset+uint16(i)] = b
		}
	}
}

func (c *codeList) size() uint16 {
	slice := *c.lst
	last := slice[len(slice)-1]
	j := uint16(len(last.val)) + last.offset
	return j - c.startoffset
}

func (c *codeList) addLabel(name string, li int) {
	c.add(name, 0)
	slice := *c.lst
	slice[len(slice)-1].li = li
}

func (c *codeList) addString(s string) {
	s = s[1 : len(s)-1]
	c.add("STRING", 0)
	slice := *c.lst
	res := []byte{}
	res = append(res, byte(len(s)))
	res = append(res, []byte(s)...)
	if len(res)%CELLL == (CELLL / 2) {
		res = append(res, byte(0))
	}
	slice[len(slice)-1].val = res
}

// gives labelled items the right addresses
// which we don't do till the codelist is complete
func (c *codeList) fixLabels() {
	slice := *c.lst
	for _, val := range slice {
		if val.li != -1 {
			off := slice[val.li].offset
			binary.LittleEndian.PutUint16(val.val, off)
		}
	}

}

func (f *Forth) compileWords(name string, words []string, labels map[string]uint16, bitmask int) (err error) {
	err = nil
	startaddr := CODEE + (CELLL * f.prims)
	codelist := codeList{&[]codeitem{}, startaddr}
	possible := []struct {
		mtype  string
		method func(w string) error
	}{
		{"PRIM", func(w string) error {
			if addr, e := f.Addr(w); e == nil {
				codelist.add(w, addr)
				return nil
			} else {
				return e
			}
		}},
		{"NUM", func(w string) error {
			if addr, e := strconv.ParseInt(w, 0, 0); e == nil {
				codelist.add(w, uint16(addr))
				return nil
			} else {
				return e
			}
		}},
		{"LABEL", func(w string) error {
			res := errors.New(fmt.Sprintf("No label for %s", w))
			if addr, ok := labels[w]; ok {
				codelist.addLabel(w, int(addr))
				return nil
			}
			return res
		}},
		{"ASM", func(w string) error {
			res := errors.New(fmt.Sprintf("ERROR: No word corresponds to %s", w))
			if name, ok := asm2forth[w]; ok {
				if addr, e := f.Addr(name); e == nil {
					codelist.add(w, addr)
					return e
				} else {
					return e
				}
			}
			return res
		}},
		{"CHR", func(w string) error {
			res := errors.New(fmt.Sprintf("could not find character in %s", w))
			if strings.HasPrefix(w, "'") && strings.HasSuffix(w, "'") && len(w) == 3 {
				codelist.add(w, uint16(byte(w[1])))
				return nil
			}
			return res
		}},
		{"STRING", func(w string) error {
			res := errors.New(fmt.Sprintf("could not find string in %s", w))
			if strings.HasPrefix(w, "'") && strings.HasSuffix(w, "'") && len(w) > 3 {
				codelist.addString(w)
				return nil
			}
			return res
		}},
	}
	parseWord := func(word string) (err error) {
		for _, j := range possible {
			if err := j.method(word); err == nil {
				return nil
			}
		}
		return errors.New(fmt.Sprintf("no way to parse %s", word))
	}
	var nprims uint16
	nprims = 0
	for _, word := range words {
		if err := parseWord(word); err != nil {
			fmt.Printf("Could not add %s, defined as %v, because %v\n", name, words, err)
			return err
		}
	}
	codelist.fixLabels()
	codelist.intoForth(f)
	//codelist.println()
	nprims = codelist.size() / CELLL
	if codelist.size()%CELLL == 1 {
		fmt.Println("BUGBUG ***** odd length for colon def", codelist.size(), codelist.lst)
	}
	f.newWord(name, startaddr, bitmask)
	f.prims = f.prims + nprims
	return err
}

func (f *Forth) doUserVariables() {
	QRX, _ := f.Addr("?RX")
	TXSTO, _ := f.Addr("TX!")
	ACCEP, _ := f.Addr("accept")
	KTAP, _ := f.Addr("kTAP")
	DOTOK, _ := f.Addr(".OK")
	INTER, _ := f.Addr("$INTERPRET")
	NUMBQ, _ := f.Addr("NUMBER?")
	CTOP := CODEE + CELLL*f.prims
	NTOP := f.NP
	LASTN := f.LAST

	initvars := []uint16{0, 0, 0, 0, //reserved
		SPP,   //SP0
		RPP,   //RP0
		QRX,   //'?KEY
		TXSTO, //'EMIT
		ACCEP, //'EXPECT
		KTAP,  //'TAP
		TXSTO, //'ECHO
		DOTOK, //'PROMPT
		BASEE, //BASE
		0,     //tmp
		0,     //SPAN
		0,     //>IN
		0,     //#TIB
		TIBB,  //TIB
		0,     //CSP
		INTER, //'EVAL
		NUMBQ, //'NUMBER
		0,     //HLD
		0,     //HANDLER
		0,     //CONTEXT pointer
	}
	for i := 0; i < VOCSS; i++ {
		initvars = append(initvars, 0) //VOCSS DUP (0) vocabulary stack
	}
	therest := []uint16{
		0,     //CURRENT pointer
		0,     //vocabulary link pointer
		CTOP,  //CP
		NTOP,  //NP
		LASTN, //LAST
	}

	initvars = append(initvars, therest...)
	UZERO := uint16(0)
	f.prim2addr["UZERO"] = UZERO
	f.prim2addr["ULAST-UZERO"] = uint16(CELLL * len(initvars))
	for i, v := range initvars {
		dstp := UZERO + uint16(i)*CELLL
		f.SetWordPtr(dstp, v)
	}

}

func (f *Forth) WordFromASM(asm string) (err error) {
	words := []string{}
	labels := make(map[string]uint16)
	bitmask := 0
	name := ""
	err = nil
	restart := func() {
		words = []string{}
		labels = make(map[string]uint16)
		bitmask = 0
	}
	getname := func(line string) string {
		for _, sep := range []string{"'", `"`} {
			i := strings.Index(line, sep)
			if i == -1 {
				continue
			}
			j := strings.LastIndex(line, sep)
			if j == -1 {
				continue
			}
			if i == j {
				continue
			}
			return line[i+1 : j]
		}
		return "NOTAWORD"
	}

	inlinestring := func(line string) string {
		i := strings.Index(line, "'")
		j := strings.LastIndex(line, "'")
		spart := line[i : j+1]
		return spart
	}

	for _, line := range strings.Split(asm, "\n") {
		fields := strings.FieldsFunc(line, func(r rune) bool {
			if r == ' ' || r == ',' || r == '\t' {
				return true
			}
			return false
		})
		toks := fields
		// remove comments
		var i int
		for _, tok := range toks {
			if strings.HasPrefix(tok, ";") {
				break
			}
			i += 1
		}
		toks = toks[:i]
	tokenloop:
		for len(toks) > 0 {
			tok := toks[0]

			toks = toks[1:]
			switch {
			case tok == "$COLON":
				if name != "" {
					if err = f.compileWords(name, words, labels, bitmask); err != nil {
						return err
					}
					restart()
				}
				name = getname(line)
				vname := fields[len(fields)-1]
				if strings.Contains(line, "COMPO+") {
					bitmask |= COMPO
				}
				if strings.Contains(line, "IMEDD+") {
					bitmask |= IMEDD
				}
				asm2forth[vname] = name
				words = append(words, []string{"CALLL", "doLIST"}...)
				break tokenloop
			case tok == "$USER":
				if name != "" {
					if err = f.compileWords(name, words, labels, bitmask); err != nil {
						return err
					}
					restart()
				}
				name = toks[1]
				name = name[1 : len(name)-1]
				vname := fields[3]
				asm2forth[vname] = name
				words = append(words, []string{"CALLL", "doLIST", "doUSER", strconv.Itoa(int(f._USER))}...)
				//				fmt.Println("user variable", name, " offset is", f._USER)
				f._USER += CELLL
				if m, ok := f.macros[name]; ok {
					//					fmt.Println("running macro", m, "for word", name)
					m()
				}
				break tokenloop
			case strings.HasSuffix(tok, ":") && tok == fields[0]:
				label := tok
				label = label[:len(label)-1]
				labels[label] = uint16(len(words))
			case tok == "DW":
				words = append(words, toks...)
				break tokenloop
			case tok == "D$":
				words = append(words, toks[0])
				words = append(words, inlinestring(line))
				break tokenloop
			}
		}
	}
	f.doUserVariables()
	err = f.compileWords(name, words, labels, bitmask)
	f.doUserVariables()
	return err

}

func (f *Forth) addHiforth() {
	constants := []struct {
		s string
		v uint16
	}{
		{"UPP", UPP},
		{"CELLL", CELLL},
		{"0-CELLL", asuint16(int16(0 - CELLL))},
		{"CRR", 13},
		{"ERR", 27},
		{"BASEE", 10},
		{"VOCSS", VOCSS},
		{"MASKK", MASKK},
		{"LF", 10},
		{"BKSPP", 8},
		{"TIC", 39},
		{"COMPO", COMPO},
		{"IMEDD", IMEDD},
		{"TIBB", TIBB},
		{"RPP", RPP},
		{"EM", EM},
		{"COLDD", COLDD},
		{"SPP", SPP},
		{"NAMEE", NAMEE},
		{"CODEE", CODEE},
		{"CALLL", CALLL},
		{"VERSION", VERSION},
	}

	for _, v := range constants {
		f.prim2addr[v.s] = v.v
	}
	asm2forth = make(map[string]string)
	f.macros = make(map[string]fn)
	f.macros["#TIB"] = func() {
		f._USER = f._USER + CELLL
	}
	f.macros["CONTEXT"] = func() {
		f._USER = f._USER + VOCSS*CELLL
	}
	f.macros["CURRENT"] = func() {
		f._USER = f._USER + CELLL
	}

	amap := []struct {
		aword string
		fword string
	}{
		{"QRX", "?RX"},
		{"TXSTO", "TX!"},
		{"STOIO", "!IO"},
		{"DOLIT", "doLIT"},
		{"DOLST", "doLIST"},
		{"EXIT", "EXIT"},
		{"EXECU", "EXECUTE"},
		{"DONXT", "next"},
		{"QBRAN", "?branch"},
		{"BRAN", "branch"},
		{"STORE", "!"},
		{"AT", "@"},
		{"CSTOR", "C!"},
		{"CAT", "C@"},
		{"RPAT", "RP@"},
		{"RPSTO", "RP!"},
		{"RFROM", "R>"},
		{"RAT", "R@"},
		{"TOR", ">R"},
		{"SPAT", "SP@"},
		{"SPSTO", "SP!"},
		{"DROP", "DROP"},
		{"DUPP", "DUP"},
		{"SWAP", "SWAP"},
		{"OVER", "OVER"},
		{"ZLESS", "0<"},
		{"ANDD", "AND"},
		{"ORR", "OR"},
		{"XORR", "XOR"},
		{"UPLUS", "UM+"},
		{"TXSTO", "TX!"},
		{"STOIO", "!IO"},
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

;   CSP		( -- a )
;		Hold the stack pointer for error checking.

		$USER	3,'CSP',CSP

;   'EVAL	( -- a )
;		Execution vector of EVAL.

		$USER	5,"'EVAL",TEVAL

;   'NUMBER	( -- a )
;		Execution vector of NUMBER?.

		$USER	7,"'NUMBER",TNUMB

;   HLD		( -- a )
;		Hold a pointer in building a numeric output string.

		$USER	3,'HLD',HLD

;   HANDLER	( -- a )
;		Hold the return stack pointer for error handling.

		$USER	7,'HANDLER',HANDL

;   CONTEXT	( -- a )
;		A area to specify vocabulary search order.

		$USER	7,'CONTEXT',CNTXT
		_USER = _USER+VOCSS*CELLL	;vocabulary stack

;   CURRENT	( -- a )
;		Point to the vocabulary to be extended.

		$USER	7,'CURRENT',CRRNT
		_USER = _USER+CELLL		;vocabulary link pointer

;   CP		( -- a )
;		Point to the top of the code dictionary.

		$USER	2,'CP',CP

;   NP		( -- a )
;		Point to the bottom of the name dictionary.

		$USER	2,'NP',NP

;   LAST	( -- a )
;		Point to the last name in the name dictionary.

		$USER	4,'LAST',LAST

;; Common functions

;   doVOC	( -- )
;		Run time action of VOCABULARY's.

		$COLON	COMPO+5,'doVOC',DOVOC
		DW	RFROM,CNTXT,STORE,EXIT

;   FORTH	( -- )
;		Make FORTH the context vocabulary.

		$COLON	5,'FORTH',FORTH
		DW	DOVOC
		DW	0			;vocabulary head pointer
		DW	0			;vocabulary link pointer

;   ?DUP	( w -- w w | 0 )
;		_Dup tos if its is not zero.

		$COLON	4,'?DUP',QDUP
		DW	DUPP
		DW	QBRAN,QDUP1
		DW	DUPP
QDUP1:		DW	EXIT

;   ROT		( w1 w2 w3 -- w2 w3 w1 )
;		Rot 3rd item to top.

		$COLON	3,'ROT',ROT
		DW	TOR,SWAP,RFROM,SWAP,EXIT

;   2DROP	( w w -- )
;		Discard two items on stack.

		$COLON	5,'2DROP',DDROP
		DW	DROP,DROP,EXIT

;   2DUP	( w1 w2 -- w1 w2 w1 w2 )
;		_Duplicate top two items.

		$COLON	4,'2DUP',DDUP
		DW	OVER,OVER,EXIT

;   +		( w w -- sum )
;		Add top two items.

		$COLON	1,'+',PLUS
		DW	UPLUS,DROP,EXIT

;   D+		( d d -- d )
;		Double addition, as an example using UM+.
;
 		$COLON	2,'D+',DPLUS
 		DW	TOR,SWAP,TOR,UPLUS
 		DW	RFROM,RFROM,PLUS,PLUS,EXIT

;   NOT		( w -- w )
;		One's complement of tos.

		$COLON	3,'NOT',INVER
		DW	DOLIT,-1,XORR,EXIT

;   NEGATE	( n -- -n )
;		Two's complement of tos.

		$COLON	6,'NEGATE',NEGAT
		DW	INVER,DOLIT,1,PLUS,EXIT

;   DNEGATE	( d -- -d )
;		Two's complement of top double.

		$COLON	7,'DNEGATE',DNEGA
		DW	INVER,TOR,INVER
		DW	DOLIT,1,UPLUS
		DW	RFROM,PLUS,EXIT

;   -		( n1 n2 -- n1-n2 )
;		Subtraction.

		$COLON	1,'-',SUBB
		DW	NEGAT,PLUS,EXIT

;   ABS		( n -- n )
;		Return the absolute value of n.

		$COLON	3,'ABS',ABSS
		DW	DUPP,ZLESS
		DW	QBRAN,ABS1
		DW	NEGAT
ABS1:		DW	EXIT

;   =		( w w -- t )
;		Return true if top two are equal.

		$COLON	1,'=',EQUAL
		DW	XORR
		DW	QBRAN,EQU1
		DW	DOLIT,0,EXIT		;false flag
EQU1:		DW	DOLIT,-1,EXIT		;true flag

;   U<		( u u -- t )
;		Unsigned compare of top two items.

		$COLON	2,'U<',ULESS
		DW	DDUP,XORR,ZLESS
		DW	QBRAN,ULES1
		DW	SWAP,DROP,ZLESS,EXIT
ULES1:		DW	SUBB,ZLESS,EXIT

;   <		( n1 n2 -- t )
;		Signed compare of top two items.

		$COLON	1,'<',LESS
		DW	DDUP,XORR,ZLESS
		DW	QBRAN,LESS1
		DW	DROP,ZLESS,EXIT
LESS1:		DW	SUBB,ZLESS,EXIT

;   MAX		( n n -- n )
;		Return the greater of two top stack items.

		$COLON	3,'MAX',MAX
		DW	DDUP,LESS
		DW	QBRAN,MAX1
		DW	SWAP
MAX1:		DW	DROP,EXIT

;   MIN		( n n -- n )
;		Return the smaller of top two stack items.

		$COLON	3,'MIN',MIN
		DW	DDUP,SWAP,LESS
		DW	QBRAN,MIN1
		DW	SWAP
MIN1:		DW	DROP,EXIT

;   WITHIN	( u ul uh -- t )
;		Return true if u is within the range of ul and uh.

		$COLON	6,'WITHIN',WITHI
		DW	OVER,SUBB,TOR			;ul <= u < uh
		DW	SUBB,RFROM,ULESS,EXIT

;; Divide

;   UM/MOD	( udl udh u -- ur uq )
;		Unsigned divide of a double by a single. Return mod and quotient.

		$COLON	6,'UM/MOD',UMMOD
		DW	DDUP,ULESS
		DW	QBRAN,UMM4
		DW	NEGAT,DOLIT,15,TOR
UMM1:		DW	TOR,DUPP,UPLUS
		DW	TOR,TOR,DUPP,UPLUS
		DW	RFROM,PLUS,DUPP
		DW	RFROM,RAT,SWAP,TOR
		DW	UPLUS,RFROM,ORR
		DW	QBRAN,UMM2
		DW	TOR,DROP,DOLIT,1,PLUS,RFROM
		DW	BRAN,UMM3
UMM2:		DW	DROP
UMM3:		DW	RFROM
		DW	DONXT,UMM1
		DW	DROP,SWAP,EXIT
UMM4:		DW	DROP,DDROP
		DW	DOLIT,-1,DUPP,EXIT	;overflow, return max

;   M/MOD	( d n -- r q )
;		Signed floored divide of double by single. Return mod and quotient.

		$COLON	5,'M/MOD',MSMOD
		DW	DUPP,ZLESS,DUPP,TOR
		DW	QBRAN,MMOD1
		DW	NEGAT,TOR,DNEGA,RFROM
MMOD1:		DW	TOR,DUPP,ZLESS
		DW	QBRAN,MMOD2
		DW	RAT,PLUS
MMOD2:		DW	RFROM,UMMOD,RFROM
		DW	QBRAN,MMOD3
		DW	SWAP,NEGAT,SWAP
MMOD3:		DW	EXIT

;   /MOD	( n n -- r q )
;		Signed divide. Return mod and quotient.

		$COLON	4,'/MOD',SLMOD
		DW	OVER,ZLESS,SWAP,MSMOD,EXIT

;   MOD		( n n -- r )
;		Signed divide. Return mod only.

		$COLON	3,'MOD',MODD
		DW	SLMOD,DROP,EXIT

;   /		( n n -- q )
;		Signed divide. Return quotient only.

		$COLON	1,'/',SLASH
		DW	SLMOD,SWAP,DROP,EXIT

;; Multiply

;   UM*		( u u -- ud )
;		Unsigned multiply. Return double product.

		$COLON	3,'UM*',UMSTA
		DW	DOLIT,0,SWAP,DOLIT,15,TOR
UMST1:		DW	DUPP,UPLUS,TOR,TOR
		DW	DUPP,UPLUS,RFROM,PLUS,RFROM
		DW	QBRAN,UMST2
		DW	TOR,OVER,UPLUS,RFROM,PLUS
UMST2:		DW	DONXT,UMST1
		DW	ROT,DROP,EXIT

;   *		( n n -- n )
;		Signed multiply. Return single product.

		$COLON	1,'*',STAR
		DW	UMSTA,DROP,EXIT

;   M*		( n n -- d )
;		Signed multiply. Return double product.

		$COLON	2,'M*',MSTAR
		DW	DDUP,XORR,ZLESS,TOR
		DW	ABSS,SWAP,ABSS,UMSTA
		DW	RFROM
		DW	QBRAN,MSTA1
		DW	DNEGA
MSTA1:		DW	EXIT

;   */MOD	( n1 n2 n3 -- r q )
;		Multiply n1 and n2, then divide by n3. Return mod and quotient.

		$COLON	5,'*/MOD',SSMOD
		DW	TOR,MSTAR,RFROM,MSMOD,EXIT

;   */		( n1 n2 n3 -- q )
;		Multiply n1 by n2, then divide by n3. Return quotient only.

		$COLON	2,'*/',STASL
		DW	SSMOD,SWAP,DROP,EXIT

;; Miscellaneous

;   CELL+	( a -- a )
;		Add cell size in byte to address.

		$COLON	5,'CELL+',CELLP
		DW	DOLIT,CELLL,PLUS,EXIT

;   CELL-	( a -- a )
;		Subtract cell size in byte from address.

		$COLON	5,'CELL-',CELLM
		DW	DOLIT,0-CELLL,PLUS,EXIT

;   CELLS	( n -- n )
;		Multiply tos by cell size in bytes.

		$COLON	5,'CELLS',CELLS
		DW	DOLIT,CELLL,STAR,EXIT

;   ALIGNED	( b -- a )
;		Align address to the cell boundary.

		$COLON	7,'ALIGNED',ALGND
		DW	DUPP,DOLIT,0,DOLIT,CELLL
		DW	UMMOD,DROP,DUPP
		DW	QBRAN,ALGN1
		DW	DOLIT,CELLL,SWAP,SUBB
ALGN1:		DW	PLUS,EXIT

;   BL		( -- 32 )
;		Return 32, the blank character.

		$COLON	2,'BL',BLANK
		DW	DOLIT,32,EXIT

;   >CHAR	( c -- c )
;		Filter non-printing characters.

		$COLON	5,'>CHAR',TCHAR
		DW	DOLIT,0x07F,ANDD,DUPP	;mask msb
		DW	DOLIT,127,BLANK,WITHI	;check for printable
		DW	QBRAN,TCHA1
		DW	DROP,DOLIT,'_'		;replace non-printables
TCHA1:		DW	EXIT

;   DEPTH	( -- n )
;		Return the depth of the data stack.

		$COLON	5,'DEPTH',DEPTH
		DW	SPAT,SZERO,AT,SWAP,SUBB
		DW	DOLIT,CELLL,SLASH,EXIT

;   PICK	( ... +n -- ... w )
;		Copy the nth stack item to tos.

		$COLON	4,'PICK',PICK
		DW	DOLIT,1,PLUS,CELLS
		DW	SPAT,PLUS,AT,EXIT

;; Memory access

;   +!		( n a -- )
;		Add n to the contents at address a.

		$COLON	2,'+!',PSTOR
		DW	SWAP,OVER,AT,PLUS
		DW	SWAP,STORE,EXIT

;   2!		( d a -- )
;		Store the double integer to address a.

		$COLON	2,'2!',DSTOR
		DW	SWAP,OVER,STORE
		DW	CELLP,STORE,EXIT

;   2@		( a -- d )
;		Fetch double integer from address a.

		$COLON	2,'2@',DAT
		DW	DUPP,CELLP,AT
		DW	SWAP,AT,EXIT

;   COUNT	( b -- b +n )
;		Return count byte of a string and add 1 to byte address.

		$COLON	5,'COUNT',COUNT
		DW	DUPP,DOLIT,1,PLUS
		DW	SWAP,CAT,EXIT

;   HERE	( -- a )
;		Return the top of the code dictionary.

		$COLON	4,'HERE',HERE
		DW	CP,AT,EXIT

;   PAD		( -- a )
;		Return the address of a temporary buffer.

		$COLON	3,'PAD',PAD
		DW	HERE,DOLIT,80,PLUS,EXIT

;   TIB		( -- a )
;		Return the address of the terminal input buffer.

		$COLON	3,'TIB',TIB
		DW	NTIB,CELLP,AT,EXIT

;   @EXECUTE	( a -- )
;		_Execute vector stored in address a.

		$COLON	8,'@EXECUTE',ATEXE
		DW	AT,QDUP			;?address or zero
		DW	QBRAN,EXE1
		DW	EXECU			;execute if non-zero
EXE1:		DW	EXIT			;do nothing if zero

;   CMOVE	( b1 b2 u -- )
;		Copy u bytes from b1 to b2.

		$COLON	5,'CMOVE',CMOVE
		DW	TOR
		DW	BRAN,CMOV2
CMOV1:		DW	TOR,DUPP,CAT
		DW	RAT,CSTOR
		DW	DOLIT,1,PLUS
		DW	RFROM,DOLIT,1,PLUS
CMOV2:		DW	DONXT,CMOV1
		DW	DDROP,EXIT

;   FILL	( b u c -- )
;		Fill u bytes of character c to area beginning at b.

		$COLON	4,'FILL',FILL
		DW	SWAP,TOR,SWAP
		DW	BRAN,FILL2
FILL1:		DW	DDUP,CSTOR,DOLIT,1,PLUS
FILL2:		DW	DONXT,FILL1
		DW	DDROP,EXIT

;   -TRAILING	( b u -- b u )
;		Adjust the count to eliminate trailing white space.

		$COLON	9,'-TRAILING',DTRAI
		DW	TOR
		DW	BRAN,DTRA2
DTRA1:		DW	BLANK,OVER,RAT,PLUS,CAT,LESS
		DW	QBRAN,DTRA2
		DW	RFROM,DOLIT,1,PLUS,EXIT	;adjusted count
DTRA2:		DW	DONXT,DTRA1
		DW	DOLIT,0,EXIT		;count=0

;   PACK$	( b u a -- a )
;		Build a counted string with u characters from b. Null fill.

		$COLON	5,'PACK$',PACKS
		DW	ALGND,DUPP,TOR		;strings only on cell boundary
		DW	OVER,DUPP,DOLIT,0
		DW	DOLIT,CELLL,UMMOD,DROP	;count mod cell
		DW	SUBB,OVER,PLUS
		DW	DOLIT,0,SWAP,STORE	;null fill cell
		DW	DDUP,CSTOR,DOLIT,1,PLUS	;save count
		DW	SWAP,CMOVE,RFROM,EXIT	;move string

;; Numeric output, single precision

;   DIGIT	( u -- c )
;		Convert digit u to a character.

		$COLON	5,'DIGIT',DIGIT
		DW	DOLIT,9,OVER,LESS
		DW	DOLIT,7,ANDD,PLUS
		DW	DOLIT,'0',PLUS,EXIT

;   EXTRACT	( n base -- n c )
;		Extract the least significant digit from n.

		$COLON	7,'EXTRACT',EXTRC
		DW	DOLIT,0,SWAP,UMMOD
		DW	SWAP,DIGIT,EXIT

;   <#		( -- )
;		Initiate the numeric output process.

		$COLON	2,'<#',BDIGS
		DW	PAD,HLD,STORE,EXIT

;   HOLD	( c -- )
;		Insert a character into the numeric output string.

		$COLON	4,'HOLD',HOLD
		DW	HLD,AT,DOLIT,1,SUBB
		DW	DUPP,HLD,STORE,CSTOR,EXIT

;   #		( u -- u )
;		Extract one digit from u and append the digit to output string.

		$COLON	1,'#',DIG
		DW	BASE,AT,EXTRC,HOLD,EXIT

;   #S		( u -- 0 )
;		Convert u until all digits are added to the output string.

		$COLON	2,'#S',DIGS
DIGS1:		DW	DIG,DUPP
		DW	QBRAN,DIGS2
		DW	BRAN,DIGS1
DIGS2:		DW	EXIT

;   SIGN	( n -- )
;		Add a minus sign to the numeric output string.

		$COLON	4,'SIGN',SIGN
		DW	ZLESS
		DW	QBRAN,SIGN1
		DW	DOLIT,'-',HOLD
SIGN1:		DW	EXIT

;   #>		( w -- b u )
;		Prepare the output string to be TYPE'd.

		$COLON	2,'#>',EDIGS
		DW	DROP,HLD,AT
		DW	PAD,OVER,SUBB,EXIT

;   str		( n -- b u )
;		Convert a signed integer to a numeric string.

		$COLON	3,'str',STR
		DW	DUPP,TOR,ABSS
		DW	BDIGS,DIGS,RFROM
		DW	SIGN,EDIGS,EXIT

;   HEX		( -- )
;		Use radix 16 as base for numeric conversions.

		$COLON	3,'HEX',HEX
		DW	DOLIT,16,BASE,STORE,EXIT

;   DECIMAL	( -- )
;		Use radix 10 as base for numeric conversions.

		$COLON	7,'DECIMAL',DECIM
		DW	DOLIT,10,BASE,STORE,EXIT

;; Numeric input, single precision

;   DIGIT?	( c base -- u t )
;		Convert a character to its numeric value. A flag indicates success.

		$COLON	6,'DIGIT?',DIGTQ
		DW	TOR,DOLIT,'0',SUBB
		DW	DOLIT,9,OVER,LESS
		DW	QBRAN,DGTQ1
		DW	DOLIT,7,SUBB
		DW	DUPP,DOLIT,10,LESS,ORR
DGTQ1:		DW	DUPP,RFROM,ULESS,EXIT

;   NUMBER?	( a -- n T | a F )
;		Convert a number string to integer. Push a flag on tos.

		$COLON	7,'NUMBER?',NUMBQ
		DW	BASE,AT,TOR,DOLIT,0,OVER,COUNT
		DW	OVER,CAT,DOLIT,'$',EQUAL
		DW	QBRAN,NUMQ1
		DW	HEX,SWAP,DOLIT,1,PLUS
		DW	SWAP,DOLIT,1,SUBB
NUMQ1:		DW	OVER,CAT,DOLIT,'-',EQUAL,TOR
		DW	SWAP,RAT,SUBB,SWAP,RAT,PLUS,QDUP
		DW	QBRAN,NUMQ6
		DW	DOLIT,1,SUBB,TOR
NUMQ2:		DW	DUPP,TOR,CAT,BASE,AT,DIGTQ
		DW	QBRAN,NUMQ4
		DW	SWAP,BASE,AT,STAR,PLUS,RFROM
		DW	DOLIT,1,PLUS
		DW	DONXT,NUMQ2
		DW	RAT,SWAP,DROP
		DW	QBRAN,NUMQ3
		DW	NEGAT
NUMQ3:		DW	SWAP
		DW	BRAN,NUMQ5
NUMQ4:		DW	RFROM,RFROM,DDROP,DDROP,DOLIT,0
NUMQ5:		DW	DUPP
NUMQ6:		DW	RFROM,DDROP
		DW	RFROM,BASE,STORE,EXIT

;; Basic I/O

;   ?KEY	( -- c T | F )
;		Return input character and true, or a false if no input.

		$COLON	4,'?KEY',QKEY
		DW	TQKEY,ATEXE,EXIT

;   KEY		( -- c )
;		Wait for and return an input character.

		$COLON	3,'KEY',KEY
KEY1:		DW	QKEY
		DW	QBRAN,KEY1
		DW	EXIT

;   EMIT	( c -- )
;		Send a character to the output device.

		$COLON	4,'EMIT',EMIT
		DW	TEMIT,ATEXE,EXIT

;   NUF?	( -- t )
;		Return false if no input, else pause and if CR return true.

		$COLON	4,'NUF?',NUFQ
		DW	QKEY,DUPP
		DW	QBRAN,NUFQ1
		DW	DDROP,KEY,DOLIT,CRR,EQUAL
NUFQ1:		DW	EXIT

;   PACE	( -- )
;		Send a pace character for the file downloading process.

		$COLON	4,'PACE',PACE
		DW	DOLIT,11,EMIT,EXIT

;   SPACE	( -- )
;		Send the blank character to the output device.

		$COLON	5,'SPACE',SPACE
		DW	BLANK,EMIT,EXIT

;   SPACES	( +n -- )
;		Send n spaces to the output device.

		$COLON	6,'SPACES',SPACS
		DW	DOLIT,0,MAX,TOR
		DW	BRAN,CHAR2
CHAR1:		DW	SPACE
CHAR2:		DW	DONXT,CHAR1
		DW	EXIT

;   TYPE	( b u -- )
;		Output u characters from b.

		$COLON	4,'TYPE',TYPEE
		DW	TOR
		DW	BRAN,TYPE2
TYPE1:		DW	DUPP,CAT,EMIT
		DW	DOLIT,1,PLUS
TYPE2:		DW	DONXT,TYPE1
		DW	DROP,EXIT

;   CR		( -- )
;		Output a carriage return and a line feed.

		$COLON	2,'CR',CR
		DW	DOLIT,CRR,EMIT
		DW	DOLIT,LF,EMIT,EXIT

;   do$		( -- a )
;		Return the address of a compiled string.

		$COLON	COMPO+3,'do$',DOSTR
		DW	RFROM,RAT,RFROM,COUNT,PLUS
		DW	ALGND,TOR,SWAP,TOR,EXIT

;   $"|		( -- a )
;		Run time routine compiled by $". Return address of a compiled string.

		$COLON	COMPO+3,'$"|',STRQP
		DW	DOSTR,EXIT		;force a call to do$

;   ."|		( -- )
;		Run time routine of ." . Output a compiled string.

		$COLON	COMPO+3,'."|',DOTQP
		DW	DOSTR,COUNT,TYPEE,EXIT

;   .R		( n +n -- )
;		Display an integer in a field of n columns, right justified.

		$COLON	2,'.R',DOTR
		DW	TOR,STR,RFROM,OVER,SUBB
		DW	SPACS,TYPEE,EXIT

;   U.R		( u +n -- )
;		Display an unsigned integer in n column, right justified.

		$COLON	3,'U.R',UDOTR
		DW	TOR,BDIGS,DIGS,EDIGS
		DW	RFROM,OVER,SUBB
		DW	SPACS,TYPEE,EXIT

;   U.		( u -- )
;		Display an unsigned integer in free format.

		$COLON	2,'U.',UDOT
		DW	BDIGS,DIGS,EDIGS
		DW	SPACE,TYPEE,EXIT

;   .		( w -- )
;		Display an integer in free format, preceeded by a space.

		$COLON	1,'.',DOT
		DW	BASE,AT,DOLIT,10,XORR	;?decimal
		DW	QBRAN,DOT1
		DW	UDOT,EXIT		;no, display unsigned
DOT1:		DW	STR,SPACE,TYPEE,EXIT	;yes, display signed

;   ?		( a -- )
;		Display the contents in a memory cell.

		$COLON	1,'?',QUEST
		DW	AT,DOT,EXIT

;; Parsing

;   parse	( b u c -- b u delta ; <string> )
;		Scan string delimited by c. Return found string and its offset.

		$COLON	5,'parse',PARS
		DW	TEMP,STORE,OVER,TOR,DUPP
		DW	QBRAN,PARS8
		DW	DOLIT,1,SUBB,TEMP,AT,BLANK,EQUAL
		DW	QBRAN,PARS3
		DW	TOR
PARS1:		DW	BLANK,OVER,CAT		;skip leading blanks ONLY
		DW	SUBB,ZLESS,INVER
		DW	QBRAN,PARS2
		DW	DOLIT,1,PLUS
		DW	DONXT,PARS1
		DW	RFROM,DROP,DOLIT,0,DUPP,EXIT
PARS2:		DW	RFROM
PARS3:		DW	OVER,SWAP
		DW	TOR
PARS4:		DW	TEMP,AT,OVER,CAT,SUBB	;scan for delimiter
		DW	TEMP,AT,BLANK,EQUAL
		DW	QBRAN,PARS5
		DW	ZLESS
PARS5:		DW	QBRAN,PARS6
		DW	DOLIT,1,PLUS
		DW	DONXT,PARS4
		DW	DUPP,TOR
		DW	BRAN,PARS7
PARS6:		DW	RFROM,DROP,DUPP
		DW	DOLIT,1,PLUS,TOR
PARS7:		DW	OVER,SUBB
		DW	RFROM,RFROM,SUBB,EXIT
PARS8:		DW	OVER,RFROM,SUBB,EXIT

;   PARSE	( c -- b u ; <string> )
;		Scan input stream and return counted string delimited by c.

		$COLON	5,'PARSE',PARSE
		DW	TOR,TIB,INN,AT,PLUS	;current input buffer pointer
		DW	NTIB,AT,INN,AT,SUBB	;remaining count
		DW	RFROM,PARS,INN,PSTOR,EXIT

;   .(		( -- )
;		Output following string up to next ) .

		$COLON	IMEDD+2,'.(',DOTPR
		DW	DOLIT,')',PARSE,TYPEE,EXIT

;   (		( -- )
;		Ignore following string up to next ) . A comment.

		$COLON	IMEDD+1,'(',PAREN
		DW	DOLIT,')',PARSE,DDROP,EXIT

;   \		( -- )
;		Ignore following text till the end of line.

		$COLON	IMEDD+1,'\',BKSLA
		DW	NTIB,AT,INN,STORE,EXIT

;   CHAR	( -- c )
;		Parse next word and return its first character.

		$COLON	4,'CHAR',CHAR
		DW	BLANK,PARSE,DROP,CAT,EXIT

;   TOKEN	( -- a ; <string> )
;		Parse a word from input stream and copy it to name dictionary.

		$COLON	5,'TOKEN',TOKEN
		DW	BLANK,PARSE,DOLIT,31,MIN
		DW	NP,AT,OVER,SUBB,CELLM
		DW	PACKS,EXIT

;   WORD	( c -- a ; <string> )
;		Parse a word from input stream and copy it to code dictionary.

		$COLON	4,'WORD',WORDD
		DW	PARSE,HERE,PACKS,EXIT

;; Dictionary search

;   NAME>	( na -- ca )
;		Return a code address given a name address.

		$COLON	5,'NAME>',NAMET
		DW	CELLM,CELLM,AT,EXIT

;   SAME?	( a a u -- a a f \ -0+ )
;		Compare u cells in two strings. Return 0 if identical.

		$COLON	5,'SAME?',SAMEQ
		DW	TOR
		DW	BRAN,SAME2
SAME1:		DW	OVER,RAT,CELLS,PLUS,AT
		DW	OVER,RAT,CELLS,PLUS,AT
		DW	SUBB,QDUP
		DW	QBRAN,SAME2
		DW	RFROM,DROP,EXIT		;strings not equal
SAME2:		DW	DONXT,SAME1
		DW	DOLIT,0,EXIT		;strings equal

;   find	( a va -- ca na | a F )
;		Search a vocabulary for a string. Return ca and na if succeeded.

		$COLON	4,'find',FIND
		DW	SWAP,DUPP,CAT
		DW	DOLIT,CELLL,SLASH,TEMP,STORE
		DW	DUPP,AT,TOR,CELLP,SWAP
FIND1:		DW	AT,DUPP
		DW	QBRAN,FIND6
		DW	DUPP,AT,DOLIT,MASKK,ANDD,RAT,XORR
		DW	QBRAN,FIND2
		DW	CELLP,DOLIT,-1		;true flag
		DW	BRAN,FIND3
FIND2:		DW	CELLP,TEMP,AT,SAMEQ
FIND3:		DW	BRAN,FIND4
FIND6:		DW	RFROM,DROP
		DW	SWAP,CELLM,SWAP,EXIT
FIND4:		DW	QBRAN,FIND5
		DW	CELLM,CELLM
		DW	BRAN,FIND1
FIND5:		DW	RFROM,DROP,SWAP,DROP
		DW	CELLM
		DW	DUPP,NAMET,SWAP,EXIT

;   NAME?	( a -- ca na | a F )
;		Search all context vocabularies for a string.

		$COLON	5,'NAME?',NAMEQ
		DW	CNTXT,DUPP,DAT,XORR	;?context=also
		DW	QBRAN,NAMQ1
		DW	CELLM			;no, start with context
NAMQ1:		DW	TOR
NAMQ2:		DW	RFROM,CELLP,DUPP,TOR	;next in search order
		DW	AT,QDUP
		DW	QBRAN,NAMQ3
		DW	FIND,QDUP		;search vocabulary
		DW	QBRAN,NAMQ2
		DW	RFROM,DROP,EXIT		;found name
NAMQ3:		DW	RFROM,DROP		;name not found
		DW	DOLIT,0,EXIT		;false flag

;; Terminal response

;   ^H		( bot eot cur -- bot eot cur )
;		Backup the cursor by one character.

		$COLON	2,'^H',BKSP
		DW	TOR,OVER,RFROM,SWAP,OVER,XORR
		DW	QBRAN,BACK1
		DW	DOLIT,BKSPP,TECHO,ATEXE,DOLIT,1,SUBB
		DW	BLANK,TECHO,ATEXE
		DW	DOLIT,BKSPP,TECHO,ATEXE
BACK1:		DW	EXIT

;   TAP		( bot eot cur c -- bot eot cur )
;		Accept and echo the key stroke and bump the cursor.

		$COLON	3,'TAP',TAP
		DW	DUPP,TECHO,ATEXE
		DW	OVER,CSTOR,DOLIT,1,PLUS,EXIT

;   kTAP	( bot eot cur c -- bot eot cur )
;		Process a key stroke, CR or backspace.

		$COLON	4,'kTAP',KTAP
		DW	DUPP,DOLIT,CRR,XORR
		DW	QBRAN,KTAP2
		DW	DOLIT,BKSPP,XORR
		DW	QBRAN,KTAP1
		DW	BLANK,TAP,EXIT
KTAP1:		DW	BKSP,EXIT
KTAP2:		DW	DROP,SWAP,DROP,DUPP,EXIT

;   accept	( b u -- b u )
;		Accept characters to input buffer. Return with actual count.

		$COLON	6,'accept',ACCEP
		DW	OVER,PLUS,OVER
ACCP1:		DW	DDUP,XORR
		DW	QBRAN,ACCP4
		DW	KEY,DUPP
;		DW	BLANK,SUBB,DOLIT,95,ULESS
		DW	BLANK,DOLIT,127,WITHI
		DW	QBRAN,ACCP2
		DW	TAP
		DW	BRAN,ACCP3
ACCP2:		DW	TTAP,ATEXE
ACCP3:		DW	BRAN,ACCP1
ACCP4:		DW	DROP,OVER,SUBB,EXIT

;   EXPECT	( b u -- )
;		Accept input stream and store count in SPAN.

		$COLON	6,'EXPECT',EXPEC
		DW	TEXPE,ATEXE,SPAN,STORE,DROP,EXIT

;   QUERY	( -- )
;		Accept input stream to terminal input buffer.

		$COLON	5,'QUERY',QUERY
		DW	TIB,DOLIT,80,TEXPE,ATEXE,NTIB,STORE
		DW	DROP,DOLIT,0,INN,STORE,EXIT

;; Error handling

;   CATCH	( ca -- 0 | err# )
;		_Execute word at ca and set up an error frame for it.

		$COLON	5,'CATCH',CATCH
		DW	SPAT,TOR,HANDL,AT,TOR	;save error frame
		DW	RPAT,HANDL,STORE,EXECU	;execute
		DW	RFROM,HANDL,STORE	;restore error frame
		DW	RFROM,DROP,DOLIT,0,EXIT	;no error

;   THROW	( err# -- err# )
;		Reset system to current local error frame an update error flag.

		$COLON	5,'THROW',THROW
		DW	HANDL,AT,RPSTO		;restore return stack
		DW	RFROM,HANDL,STORE	;restore handler frame
		DW	RFROM,SWAP,TOR,SPSTO	;restore data stack
		DW	DROP,RFROM,EXIT

;   NULL$	( -- a )
;		Return address of a null string with zero count.

		$COLON	5,'NULL$',NULLS
		DW	DOVAR			;emulate CREATE
		DW	0
		DB	99,111,121,111,116,101
		$ALIGN

;   ABORT	( -- )
;		Reset data stack and jump to QUIT.

		$COLON	5,'ABORT',ABORT
		DW	NULLS,THROW

;   abort"	( f -- )
;		Run time routine of ABORT" . Abort with a message.

		$COLON	COMPO+6,'abort"',ABORQ
		DW	QBRAN,ABOR1		;text flag
		DW	DOSTR,THROW		;pass error string
ABOR1:		DW	DOSTR,DROP,EXIT		;drop error

;; The text interpreter

;   $INTERPRET	( a -- )
;		Interpret a word. If failed, try to convert it to an integer.

		$COLON	10,'$INTERPRET',INTER
		DW	NAMEQ,QDUP		;?defined
		DW	QBRAN,INTE1
		DW	AT,DOLIT,COMPO,ANDD	;?compile only lexicon bits
		D$	ABORQ,' compile only'
		DW	EXECU,EXIT		;execute defined word
INTE1:		DW	TNUMB,ATEXE		;convert a number
		DW	QBRAN,INTE2
		DW	EXIT
INTE2:		DW	THROW			;error

;   [		( -- )
;		Start the text interpreter.

		$COLON	IMEDD+1,'[',LBRAC
		DW	DOLIT,INTER,TEVAL,STORE,EXIT

;   .OK		( -- )
;		Display 'ok' only while interpreting.

		$COLON	3,'.OK',DOTOK
		DW	DOLIT,INTER,TEVAL,AT,EQUAL
		DW	QBRAN,DOTO1
		D$	DOTQP,' ok'
DOTO1:		DW	CR,EXIT

;   ?STACK	( -- )
;		Abort if the data stack underflows.

		$COLON	6,'?STACK',QSTAC
		DW	DEPTH,ZLESS		;check only for underflow
		D$	ABORQ,' underflow'
		DW	EXIT

;   EVAL	( -- )
;		Interpret the input stream.

		$COLON	4,'EVAL',EVAL
EVAL1:		DW	TOKEN,DUPP,CAT		;?input stream empty
		DW	QBRAN,EVAL2
		DW	TEVAL,ATEXE,QSTAC	;evaluate input, check stack
		DW	BRAN,EVAL1
EVAL2:		DW	DROP,TPROM,ATEXE,EXIT	;prompt

;; Shell

;   PRESET	( -- )
;		Reset data stack pointer and the terminal input buffer.

		$COLON	6,'PRESET',PRESE
		DW	SZERO,AT,SPSTO
		DW	DOLIT,TIBB,NTIB,CELLP,STORE,EXIT

;   xio		( a a a -- )
;		Reset the I/O vectors 'EXPECT, 'TAP, 'ECHO and 'PROMPT.

		$COLON	COMPO+3,'xio',XIO
		DW	DOLIT,ACCEP,TEXPE,DSTOR
		DW	TECHO,DSTOR,EXIT

;   FILE	( -- )
;		Select I/O vectors for file download.

		$COLON	4,'FILE',FILE
		DW	DOLIT,PACE,DOLIT,DROP
		DW	DOLIT,KTAP,XIO,EXIT

;   HAND	( -- )
;		Select I/O vectors for terminal interface.

		$COLON	4,'HAND',HAND
		DW	DOLIT,DOTOK,DOLIT,EMIT
		DW	DOLIT,KTAP,XIO,EXIT

;   I/O		( -- a )
;		Array to store default I/O vectors.

		$COLON	3,'I/O',ISLO
		DW	DOVAR			;emulate CREATE
		DW	QRX,TXSTO		;default I/O vectors

;   CONSOLE	( -- )
;		Initiate terminal interface.

		$COLON	7,'CONSOLE',CONSO
		DW	ISLO,DAT,TQKEY,DSTOR	;restore default I/O device
		DW	HAND,EXIT		;keyboard input

;   QUIT	( -- )
;		Reset return stack pointer and start text interpreter.

		$COLON	4,'QUIT',QUIT
		DW	RZERO,AT,RPSTO		;reset return stack pointer
QUIT1:		DW	LBRAC			;start interpretation
QUIT2:		DW	QUERY			;get input
		DW	DOLIT,EVAL,CATCH,QDUP	;evaluate input
		DW	QBRAN,QUIT2		;continue till error
		DW	TPROM,AT,SWAP		;save input device
		DW	CONSO,NULLS,OVER,XORR	;?display error message
		DW	QBRAN,QUIT3
		DW	SPACE,COUNT,TYPEE	;error message
		D$	DOTQP,' ? '		;error prompt
QUIT3:		DW	DOLIT,DOTOK,XORR	;?file input
		DW	QBRAN,QUIT4
		DW	DOLIT,ERR,EMIT		;file error, tell host
QUIT4:		DW	PRESE			;some cleanup
		DW	BRAN,QUIT1

;; The compiler

;   '		( -- ca )
;		Search context vocabularies for the next word in input stream.

		$COLON	1,"'",TICK
		DW	TOKEN,NAMEQ		;?defined
		DW	QBRAN,TICK1
		DW	EXIT			;yes, push code address
TICK1:		DW	THROW			;no, error

;   ALLOT	( n -- )
;		Allocate n bytes to the code dictionary.

		$COLON	5,'ALLOT',ALLOT
		DW	CP,PSTOR,EXIT		;adjust code pointer

;   ,		( w -- )
;		Compile an integer into the code dictionary.

		$COLON	1,',',COMMA
		DW	HERE,DUPP,CELLP		;cell boundary
		DW	CP,STORE,STORE,EXIT	;adjust code pointer, compile

;   [COMPILE]	( -- ; <string> )
;		Compile the next immediate word into code dictionary.

		$COLON	IMEDD+9,'[COMPILE]',BCOMP
		DW	TICK,COMMA,EXIT

;   COMPILE	( -- )
;		Compile the next address in colon list to code dictionary.

		$COLON	COMPO+7,'COMPILE',COMPI
		DW	RFROM,DUPP,AT,COMMA	;compile address
		DW	CELLP,TOR,EXIT		;adjust return address

;   LITERAL	( w -- )
;		Compile tos to code dictionary as an integer literal.

		$COLON	IMEDD+7,'LITERAL',LITER
		DW	COMPI,DOLIT,COMMA,EXIT

;   $,"		( -- )
;		Compile a literal string up to next " .

		$COLON	3,'$,"',STRCQ
		DW	DOLIT,'"',WORDD		;move string to code dictionary
		DW	COUNT,PLUS,ALGND	;calculate aligned end of string
		DW	CP,STORE,EXIT		;adjust the code pointer

;   RECURSE	( -- )
;		Make the current word available for compilation.

		$COLON	IMEDD+7,'RECURSE',RECUR
		DW	LAST,AT,NAMET,COMMA,EXIT

;; Structures

;   FOR		( -- a )
;		Start a FOR-NEXT loop structure in a colon definition.

		$COLON	IMEDD+3,'FOR',FOR
		DW	COMPI,TOR,HERE,EXIT

;   BEGIN	( -- a )
;		Start an infinite or indefinite loop structure.

		$COLON	IMEDD+5,'BEGIN',BEGIN
		DW	HERE,EXIT

;   NEXT	( a -- )
;		Terminate a FOR-NEXT loop structure.

		$COLON	IMEDD+4,'NEXT',NEXT
		DW	COMPI,DONXT,COMMA,EXIT

;   UNTIL	( a -- )
;		Terminate a BEGIN-UNTIL indefinite loop structure.

		$COLON	IMEDD+5,'UNTIL',UNTIL
		DW	COMPI,QBRAN,COMMA,EXIT

;   AGAIN	( a -- )
;		Terminate a BEGIN-AGAIN infinite loop structure.

		$COLON	IMEDD+5,'AGAIN',AGAIN
		DW	COMPI,BRAN,COMMA,EXIT

;   IF		( -- A )
;		Begin a conditional branch structure.

		$COLON	IMEDD+2,'IF',IFF
		DW	COMPI,QBRAN,HERE
		DW	DOLIT,0,COMMA,EXIT

;   AHEAD	( -- A )
;		Compile a forward branch instruction.

		$COLON	IMEDD+5,'AHEAD',AHEAD
		DW	COMPI,BRAN,HERE,DOLIT,0,COMMA,EXIT

;   REPEAT	( A a -- )
;		Terminate a BEGIN-WHILE-REPEAT indefinite loop.

		$COLON	IMEDD+6,'REPEAT',REPEA
		DW	AGAIN,HERE,SWAP,STORE,EXIT

;   THEN	( A -- )
;		Terminate a conditional branch structure.

		$COLON	IMEDD+4,'THEN',THENN
		DW	HERE,SWAP,STORE,EXIT

;   AFT		( a -- a A )
;		Jump to THEN in a FOR-AFT-THEN-NEXT loop the first time through.

		$COLON	IMEDD+3,'AFT',AFT
		DW	DROP,AHEAD,BEGIN,SWAP,EXIT

;   ELSE	( A -- A )
;		Start the false clause in an IF-ELSE-THEN structure.

		$COLON	IMEDD+4,'ELSE',ELSEE
		DW	AHEAD,SWAP,THENN,EXIT

;   WHILE	( a -- A a )
;		Conditional branch out of a BEGIN-WHILE-REPEAT loop.

		$COLON	IMEDD+5,'WHILE',WHILE
		DW	IFF,SWAP,EXIT

;   ABORT"	( -- ; <string> )
;		Conditional abort with an error message.

		$COLON	IMEDD+6,'ABORT"',ABRTQ
		DW	COMPI,ABORQ,STRCQ,EXIT

;   $"		( -- ; <string> )
;		Compile an inline string literal.

		$COLON	IMEDD+2,'$"',STRQ
		DW	COMPI,STRQP,STRCQ,EXIT

;   ."		( -- ; <string> )
;		Compile an inline string literal to be typed out at run time.

		$COLON	IMEDD+2,'."',DOTQ
		DW	COMPI,DOTQP,STRCQ,EXIT

;; Name compiler

;   ?UNIQUE	( a -- a )
;		Display a warning message if the word already exists.

		$COLON	7,'?UNIQUE',UNIQU
		DW	DUPP,NAMEQ		;?name exists
		DW	QBRAN,UNIQ1		;redefinitions are OK
		D$	DOTQP,' reDef '		;but warn the user
		DW	OVER,COUNT,TYPEE	;just in case its not planned
UNIQ1:		DW	DROP,EXIT

;   $,n		( na -- )
;		Build a new dictionary name using the string at na.

		$COLON	3,'$,n',SNAME
		DW	DUPP,CAT		;?null input
		DW	QBRAN,PNAM1
		DW	UNIQU			;?redefinition
		DW	DUPP,LAST,STORE		;save na for vocabulary link
		DW	HERE,ALGND,SWAP		;align code address
		DW	CELLM			;link address
		DW	CRRNT,AT,AT,OVER,STORE
		DW	CELLM,DUPP,NP,STORE	;adjust name pointer
		DW	STORE,EXIT		;save code pointer
PNAM1:		D$	STRQP,' name'		;null input
		DW	THROW

;; FORTH compiler

;   $COMPILE	( a -- )
;		Compile next word to code dictionary as a token or literal.

		$COLON	8,'$COMPILE',SCOMP
		DW	NAMEQ,QDUP		;?defined
		DW	QBRAN,SCOM2
		DW	AT,DOLIT,IMEDD,ANDD	;?immediate
		DW	QBRAN,SCOM1
		DW	EXECU,EXIT		;its immediate, execute
SCOM1:		DW	COMMA,EXIT		;its not immediate, compile
SCOM2:		DW	TNUMB,ATEXE		;try to convert to number
		DW	QBRAN,SCOM3
		DW	LITER,EXIT		;compile number as integer
SCOM3:		DW	THROW			;error

;   OVERT	( -- )
;		Link a new word into the current vocabulary.

		$COLON	5,'OVERT',OVERT
		DW	LAST,AT,CRRNT,AT,STORE,EXIT

;   ;		( -- )
;		Terminate a colon definition.

		$COLON	IMEDD+COMPO+1,';',SEMIS
		DW	COMPI,EXIT,LBRAC,OVERT,EXIT

;   ]		( -- )
;		Start compiling the words in the input stream.

		$COLON	1,']',RBRAC
		DW	DOLIT,SCOMP,TEVAL,STORE,EXIT

;   call,	( ca -- )
;		Assemble a call instruction to ca.

		$COLON	5,'call,',CALLC
		DW	DOLIT,CALLL,COMMA 	;Direct Threaded Code
		DW	COMMA,EXIT
; nh modified this for the golang vm

;   :		( -- ; <string> )
;		Start a new colon definition using next word as its name.

		$COLON	1,':',COLON
		DW	TOKEN,SNAME,DOLIT,DOLST
		DW	CALLC,RBRAC,EXIT

;   IMMEDIATE	( -- )
;		Make the last compiled word an immediate word.

		$COLON	9,'IMMEDIATE',IMMED
		DW	DOLIT,IMEDD,LAST,AT,AT,ORR
		DW	LAST,AT,STORE,EXIT

;; Defining words

;   USER	( u -- ; <string> )
;		Compile a new user variable.

		$COLON	4,'USER',USER
		DW	TOKEN,SNAME,OVERT
		DW	DOLIT,DOLST,CALLC
		DW	COMPI,DOUSE,COMMA,EXIT

;   CREATE	( -- ; <string> )
;		Compile a new array entry without allocating code space.

		$COLON	6,'CREATE',CREAT
		DW	TOKEN,SNAME,OVERT
		DW	DOLIT,DOLST,CALLC
		DW	COMPI,DOVAR,EXIT

;   VARIABLE	( -- ; <string> )
;		Compile a new variable initialized to 0.

		$COLON	8,'VARIABLE',VARIA
		DW	CREAT,DOLIT,0,COMMA,EXIT

;; Tools

;   _TYPE	( b u -- )
;		Display a string. Filter non-printing characters.

		$COLON	5,'_TYPE',UTYPE
		DW	TOR			;start count down loop
		DW	BRAN,UTYP2		;skip first pass
UTYP1:		DW	DUPP,CAT,TCHAR,EMIT	;display only printable
		DW	DOLIT,1,PLUS		;increment address
UTYP2:		DW	DONXT,UTYP1		;loop till done
		DW	DROP,EXIT

;   dm+		( a u -- a )
;		Dump u bytes from , leaving a+u on the stack.

		$COLON	3,'dm+',DMP
		DW	OVER,DOLIT,4,UDOTR	;display address
		DW	SPACE,TOR		;start count down loop
		DW	BRAN,PDUM2		;skip first pass
PDUM1:		DW	DUPP,CAT,DOLIT,3,UDOTR	;display numeric data
		DW	DOLIT,1,PLUS		;increment address
PDUM2:		DW	DONXT,PDUM1		;loop till done
		DW	EXIT

;   DUMP	( a u -- )
;		Dump u bytes from a, in a formatted manner.

		$COLON	4,'DUMP',DUMP
		DW	BASE,AT,TOR,HEX		;save radix, set hex
		DW	DOLIT,16,SLASH		;change count to lines
		DW	TOR			;start count down loop
DUMP1:		DW	CR,DOLIT,16,DDUP,DMP	;display numeric
		DW	ROT,ROT
		DW	SPACE,SPACE,UTYPE	;display printable characters
		DW	NUFQ,INVER		;user control
		DW	QBRAN,DUMP2
		DW	DONXT,DUMP1		;loop till done
		DW	BRAN,DUMP3
DUMP2:		DW	RFROM,DROP		;cleanup loop stack, early exit
DUMP3:		DW	DROP,RFROM,BASE,STORE	;restore radix
		DW	EXIT

;   .S		( ... -- ... )
;		Display the contents of the data stack.

		$COLON	2,'.S',DOTS
		DW	CR,DEPTH		;stack depth
		DW	TOR			;start count down loop
		DW	BRAN,DOTS2		;skip first pass
DOTS1:		DW	RAT,PICK,DOT		;index stack, display contents
DOTS2:		DW	DONXT,DOTS1		;loop till done
		D$	DOTQP,' <sp'
		DW	EXIT

;   !CSP	( -- )
;		Save stack pointer in CSP for error checking.

		$COLON	4,'!CSP',STCSP
		DW	SPAT,CSP,STORE,EXIT	;save pointer

;   ?CSP	( -- )
;		Abort if stack pointer differs from that saved in CSP.

		$COLON	4,'?CSP',QCSP
		DW	SPAT,CSP,AT,XORR	;compare pointers
		D$	ABORQ,'stacks'		;abort if different
		DW	EXIT

;   >NAME	( ca -- na | F )
;		Convert code address to a name address.

		$COLON	5,'>NAME',TNAME
		DW	CRRNT			;vocabulary link
TNAM1:		DW	CELLP,AT,QDUP		;check all vocabularies
		DW	QBRAN,TNAM4
		DW	DDUP
TNAM2:		DW	AT,DUPP			;?last word in a vocabulary
		DW	QBRAN,TNAM3
		DW	DDUP,NAMET,XORR		;compare
		DW	QBRAN,TNAM3
		DW	CELLM			;continue with next word
		DW	BRAN,TNAM2
TNAM3:		DW	SWAP,DROP,QDUP
		DW	QBRAN,TNAM1
		DW	SWAP,DROP,SWAP,DROP,EXIT
TNAM4:		DW	DROP,DOLIT,0,EXIT	;false flag

;   .ID		( na -- )
;		Display the name at address.

		$COLON	3,'.ID',DOTID
		DW	QDUP			;if zero no name
		DW	QBRAN,DOTI1
		DW	COUNT,DOLIT,0x01F,ANDD	;mask lexicon bits
		DW	UTYPE,EXIT		;display name string
DOTI1:		D$	DOTQP,' {noName}'
		DW	EXIT

;   SEE		( -- ; <string> )
;		A simple decompiler.

		$COLON	3,'SEE',SEE
		DW	TICK			;starting address
		DW	CR,CELLP
SEE1:		DW	CELLP,DUPP,AT,DUPP	;?does it contain a zero
		DW	QBRAN,SEE2
		DW	TNAME			;?is it a name
SEE2:		DW	QDUP			;name address or zero
		DW	QBRAN,SEE3
		DW	SPACE,DOTID		;display name
		DW	BRAN,SEE4
SEE3:		DW	DUPP,AT,UDOT		;display number
SEE4:		DW	NUFQ			;user control
		DW	QBRAN,SEE1
		DW	DROP,EXIT

;   WORDS	( -- )
;		Display the names in the context vocabulary.

		$COLON	5,'WORDS',WORDS
		DW	CR,CNTXT,AT		;only in context
WORS1:		DW	AT,QDUP			;?at end of list
		DW	QBRAN,WORS2
		DW	DUPP,SPACE,DOTID	;display a name
		DW	CELLM,NUFQ		;user control
		DW	QBRAN,WORS1
		DW	DROP
WORS2:		DW	EXIT

;; Hardware reset

;   VER		( -- n )
;		Return the version number of this implementation.

		$COLON	3,'VER',VERSN
		DW	DOLIT,VERSION,EXIT

;   hi		( -- )
;		Display the sign-on message of eForth.

		$COLON	2,'hi',HI
		DW	STOIO,CR		;initialize I/O
		D$	DOTQP,'eForth v'	;model
		DW	BASE,AT,HEX		;save radix
		DW	VERSN,BDIGS,DIG,DIG
		DW	DOLIT,'.',HOLD
		DW	DIGS,EDIGS,TYPEE	;format version number
		DW	BASE,STORE,CR,EXIT	;restore radix

;   'BOOT	( -- a )
;		The application startup vector.

		$COLON	5,"'BOOT",TBOOT
		DW	DOVAR
		DW	HI			;application to boot

;   COLD	( -- )
;		The hilevel cold start sequence.

		$COLON	4,'COLD',COLD
COLD1:		DW	DOLIT,UZERO,DOLIT,UPP
		DW	DOLIT,ULAST-UZERO,CMOVE	;initialize user area
		DW	PRESE			;initialize stack and TIB
		DW	TBOOT,ATEXE		;application boot
		DW	FORTH,CNTXT,AT,DUPP	;initialize search order
		DW	CRRNT,DSTOR,OVERT
		DW	QUIT			;start interpretation
		DW	BRAN,COLD1		;just in case
`,
	}

	for _, asm := range hiforth {
		err := f.WordFromASM(asm)
		if err != nil {
			fmt.Println("ERROR: ", err)
		}
	}

}
