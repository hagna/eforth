package eforth

import (
	"fmt"
	"strconv"
	"strings"
)

var asm2forth map[string]string

func (f *Forth) WordFromASM(asm string) (err error) {
	words := []string{}
	labels := make(map[string]uint16)
	name := ""
	err = nil
	for iline, line := range strings.Split(asm, "\n") {
		fields := strings.FieldsFunc(line, func(r rune) bool {
			if r == ' ' || r == ',' || r == '\t' {
				return true
			}
			return false
		})
		fmt.Println(iline)
		toks := fields
tokenloop:
		for len(toks) > 0 {
			tok := toks[0]
			toks = toks[1:]
			fmt.Println("token is", tok)
			fmt.Println("the rest is", toks)
			switch {
			case tok == "$COLON":
				name = toks[1]
				name = name[1 : len(name)-1]
				vname := fields[3]
				asm2forth[vname] = name
				words = append(words, []string{"2", ":"}...)
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
	startaddr := CODEE + (2 * f.prims)
	setit := func(i int, addr uint16) {
		f.SetWordPtr(startaddr+uint16(i*CELLL), addr)
	}
	for i, word := range words {
		fmt.Println(i, word)
		addr, e := f.Addr(word)
		b, ok := asm2forth[word]
		a, se := strconv.Atoi(word)
		c, e2 := f.Addr(b)
		v, el := labels[word]
		switch {
		case e == nil:
			setit(i, addr)
		case se == nil:
			fmt.Println("converted it to integer")
			addr = uint16(a)
			setit(i, addr)
		case el:
			fmt.Println("found it the label at", v)
			addr = v * CELLL
			fmt.Println("startaddr is", startaddr)
			fmt.Println("label addr is", addr+startaddr)
			setit(i, addr+startaddr)
		case e2 == nil && ok:
			setit(i, c)
		default:
			err = e
			fmt.Println("ERROR: could not add", name, "because", e)

		}
	}
	fmt.Println("words are", words)
	fmt.Println("hex words", dumpmem(f, startaddr, uint16(len(words)*CELLL)))
	f.prim2addr[name] = startaddr
	f.AddName(name, startaddr)
	f.prims = f.prims + 2*uint16(len(words))
	return
}

func (f *Forth) AddHiforth() {
	f.prim2addr["UPP"] = UPP
	asm2forth = make(map[string]string)
	amap := []struct {
		aword string
		fword string
	}{
		{"DOLIT", "doLIT"},
		{"QBRAN", "?BRANCH"},
		{"DUPP", "DUP"},
		{"RFROM", "R>"},
	}
	for _, v := range amap {
		asm2forth[v.aword] = v.fword
	}
	hiforth := []string{`
;   doVAR	( -- a )
;		Run time routine for VARIABLE and CREATE.

		$COLON	COMPO+5,'doVAR',DOVAR
		DW	RFROM,EXIT
`,
		`
;   UP		( -- a )
;		Pointer to the user area.

		$COLON	2,'UP',UP
		DW	DOVAR
		DW	UPP
`}

	for _, asm := range hiforth {
		err := f.WordFromASM(asm)
		if err != nil {
			fmt.Println("ERROR: ", err)
			fmt.Println(asm)
		}
	}

}
