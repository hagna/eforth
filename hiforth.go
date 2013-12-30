package eforth

import (
	"fmt"
	"strconv"
	"strings"
)

var asm2forth map[string]string

func (f* Forth) WordFromASM(asm string) (e error) {
	addrs := []uint16{}
	words := []string{}
	labels := make(map[string]uint16)
	name := ""
	isColondef := false
	e = nil
	for iline, line := range strings.Split(asm, "\n") {
		fields := strings.FieldsFunc(line, func(r rune) bool {
			if r == ' ' || r == ',' || r == '\t' {
				return true
			}
			return false
			})
		fmt.Println(fields)
        if len(fields) == 0 {
            continue
        }
        switch name {
            case "":
                fmt.Println("name blank and is", name)
                switch fields[0] {
                    case "$COLON", "$USER":
                        isColondef = true
                        name = fields[2]
                        name = name[1:len(name)-1]
                        vname := fields[3]
                        asm2forth[vname] = name	
                }
            default:
                i := 0
                if fields[0] == "DW" {
                    fmt.Println("fields[0] is DW")
                    i = 1
                }
                label := fields[0]
                if strings.HasSuffix(label, ":") {
                    label = label[:len(label)-1]
                    labels[label] = uint16(len(words))
                    if fields[1] == "DW" {
                        i = 2
                    }
                }
                if i != 0 {
                    words = append(words, fields[i:]...)
                }
                fmt.Println(iline)
                fmt.Println(words)
		}
	}
	if isColondef {
		words = append([]string{"2", ":"}, words...)
	}
	startaddr := CODEE + (2 * f.prims)
	setit := func (i  int, addr uint16) {
		f.SetWordPtr(startaddr+uint16(i*CELLL), addr)
	}
	for i, word := range words {
		fmt.Println(i, word)
		addr, e := f.Addr(word)
		if e != nil {
			b, ok := asm2forth[word]
			if !ok {	
				a, se := strconv.Atoi(word) 
				if se != nil {
					fmt.Println("labels is", labels)
					fmt.Println("word", word, "ought to be in labels")
					v, ok := labels[word]
					if !ok {
						fmt.Println("ERROR: could not add", name, "because", e)
					} else {
						fmt.Println("found it the label at", v)
						// +2 is for CALL doLIST
						addr = (v+2)*CELLL
						fmt.Println("startaddr is", startaddr)
						fmt.Println("label addr is", addr+startaddr)
						setit(i, addr+startaddr)
					}
				} else {
					fmt.Println("converted it to integer")
					addr = uint16(a)
					setit(i, addr)
				}
			} else {
				a, e2 := f.Addr(b)
				if e2 != nil {
					fmt.Println("ERROR: found", word, "in ASM map but could not add", name, "because", e2)
				} else {
					addr = uint16(a)
					setit(i, addr)	
				}
			}
		} else {
			setit(i, addr)
		}
	}
	fmt.Println("addrs is", addrs, "labels is", labels, "name is", name)
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
