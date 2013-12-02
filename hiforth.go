package eforth

import (
	"fmt"
	"strings"
)

func (f *Forth) ColonDefs() {
	all := `
: + UM+ DROP ;

: doVAR ( -- a ) R> ;

: UP ( -- a ) doVAR UPP ;

: doUSER    ( -- a, Run time routine for user variables.)
      R> @                          
      UP @ + ;                     

: doVOC ( -- ) R> CONTEXT ! ;

: FORTH ( -- ) doVOC [ 0 , 0 , ;
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
