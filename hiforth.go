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
