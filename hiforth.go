package eforth

import (
	"fmt"
	"strconv"
	"strings"
)

var asm2forth map[string]string
/*func dumpmem(f *Forth, i, k uint16) string {
	res := ""
	for _, j := range f.Memory[i:i+k] {
		res += fmt.Sprintf("%x ", j)
	}
	res += "\n"
	return res
}*/
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
		if name == "" {
			fmt.Println("name blank and is", name)
			if strings.Contains(line, "$COLON") {
				isColondef = true
				name = fields[2]
				name = name[1:len(name)-1]
				fmt.Println("name is", name)
			}
		} else {
			if len(fields) >= 1 {
			i := 0
			if fields[0] == "DW" {
				fmt.Println("fields[0] is DW")
				i = 1
			}
			label := fields[0]
			if strings.HasSuffix(label, ":") {
				labels[label] = uint16(len(words)*CELLL)
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
					v, ok := labels[word]
					if !ok {
						fmt.Println("ERROR: could not add", name, "because", e)
					} else {
						addr = v
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
	asm2forth = make(map[string]string)
	amap := []struct {
		aword string
		fword string
	}{
		{"DOLIT", "doLIT"},
	}
	for _, v := range amap {
		asm2forth[v.aword] = v.fword
	}
	all := `
: + UM+ DROP ;

: doVAR ( -- a ) R> ;

: UP ( -- a ) doVAR UPP ;

: doUSER    ( -- a, Run time routine for user variables.)
      R> @                          
      UP @ + ;                     

: doVOC ( -- ) R> CONTEXT ! ;

: FORTH ( -- ) doVOC [ 0 , 0 , ;

: ?DUP ( w -- w w | 0 ) DUP IF DUP THEN ;

: ROT ( w1 w2 w3 -- w2 w3 w1 ) >R SWAP R> SWAP ;

: 2DROP ( w w  -- ) DROP DROP ;

: 2DUP ( w1 w2 -- w1 w2 w1 w2 ) OVER OVER  ;

: NOT ( w -- w ) doLIT -1 XOR  ;

: NEGATE ( n -- -n ) NOT 1 + ;

: DNEGATE ( d -- -d ) NOT >R NOT 1 UM+ R> + ;

: D+ ( d d -- d ) >R SWAP >R UM+ R> R> + + ;

: - ( w w -- w ) NEGATE + ;

: ABS ( n -- +n ) DUP 0< IF NEGATE THEN ;

: = ( w w -- t ) XOR IF doLIT 0 EXIT THEN doLIT -1 ;

: U< ( u u -- t ) 2DUP XOR 0< IF SWAP DROP 0< EXIT THEN - 0< ;

: < ( n n -- t ) 2DUP XOR 0< IF      DROP 0< EXIT THEN - 0< ;

: MAX ( n n -- n ) 2DUP      < IF SWAP THEN DROP ;

: MIN ( n n -- n ) 2DUP SWAP < IF SWAP THEN DROP ;

: WITHIN ( u ul uh -- t  \ ul <= u < uh )
  OVER - >R - R> U< ;
`
	fmt.Println("HIFORTH")
	for _, def := range strings.Split(all, "\n\n") {
		err := f.AddWord(def)
		if err != nil {
			fmt.Println("ERROR: ", err)
			fmt.Println(def)
		}
	}

}
