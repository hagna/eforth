package eforth

import (
	"fmt"
	"io"
	"time"
)

func (f *Forth) addPrimitives() {
	words := []struct {
		word  string
		m     fn
		flags int
	}{
		{"BYE", f._BYE, 0},
		{"CALL", f._Call, 0},
		{"doLIST", f.doLIST, COMPO},
		{"!IO", f._B_IO, 0},
		{"?RX", f._Q_RX, 0},
		{"TX!", f._B_TX, 0},
		{"EXECUTE", f._Execute, 0},
		{"doLIT", f.doLIT, COMPO},
		{"EXIT", f._EXIT, 0},
		{"next", f._Next, COMPO},
		{"?branch", f._Q_branch, COMPO},
		{"branch", f._Branch, COMPO},
		{"!", f._Bang, 0},
		{"@", f._At, 0},
		{"C!", f._Cbang, 0},
		{"C@", f._Cat, 0},
		{"RP@", f._RPat, 0},
		{"RP!", f._RPbang, COMPO},
		{"R>", f._Rfrom, 0},
		{"R@", f._Rat, 0},
		{">R", f._Tor, COMPO},
		{"DROP", f._Drop, 0},
		{"DUP", f._Dup, 0},
		{"SWAP", f._Swap, 0},
		{"OVER", f._Over, 0},
		{"SP@", f._Sp_at, 0},
		{"SP!", f._Sp_bang, 0},
		{"0<", f._Zless, 0},
		{"AND", f._And, 0},
		{"OR", f._Or, 0},
		{"XOR", f._Xor, 0},
		{"UM+", f._UMplus, 0},
	}
	for _, v := range words {
		f.AddPrim(v.word, v.m, v.flags)
	}
}

/*
CODE BYE    ( -- , exit Forth )
      INT   020H                    \ return to DOS
*/
func (f *Forth) _BYE() {
	f.IP = 0xffff
}

/*
CODE  !IO   ( -- )                  \ Initialize the serial I/O devices.
      $Next
*/
// initialize IO
func (f *Forth) _B_IO() {
	//in := f.b_input
	c := make(chan uint16)
	go func() {
		buf := make([]byte, 1)
	forloop:
		for {
			if f.Input != nil {
				_, err := f.Input.Read(buf)
				//b, err := in.ReadByte()
				if err != nil {
					//            fmt.Println("could not read Byte", err)
					c <- 0
					if err == io.EOF {
						fmt.Println("break for io.EOF")
						break forloop
					}
				} else {
					b := buf[0]
					if b == 10 {
						b = 13
					}
					//            fmt.Println("\nRX:", b, string(b))
					c <- uint16(b)
					c <- asuint16(-1)
				}
			} else {
				c <- 0
			}

		}
		fmt.Println("exiting goroutine")
	}()
	f.rxchan = c
	f.Next()
}

/*
CODE  EXECUTE     ( ca -- )         \ _Execute the word at ca.
      POP   BX
      JMP   BX                      \ jump to the code address
*/
func (f *Forth) _Execute() {
	bx := f.Pop()
	f.WP = bx
}

/*
CODE  ?RX   ( -- c T | F )          \ Return input character and true,
                                    \ or a false if no input.
      $CODE 3,'?RX',QRX
      XOR   BX,BX                   \ BX=0 setup for false flag
      MOV   DL,0FFH                 \ input command
      MOV   AH,6                    \ MS-DOS Direct Console I/O
      INT   021H
      JZ    QRX3                    \ ?key ready
      OR    AL,AL                   \ AL=0 if extended char
      JNZ   QRX1                    \ ?extended character code
      INT   021H
      MOV   BH,AL                   \ extended code in msb
      JMP   QRX2
QRX1: MOV   BL,AL
QRX2: PUSH  BX                      \ save character
      MOV   BX,-1                   \ true flag
QRX3: PUSH  BX
      $Next
*/
// RX may need to be non-blocking receive
// returns either false or char true
func (f *Forth) _Q_RX() {
	c := f.rxchan
	select {
	case res := <-c:
		if res == 0 {
			f.Push(0)
		} else {
			f.Push(res)
			f.Push(<-c)
		}
	// I really don't like this solution that much
	// but at least this is a solution.
	// evidently unix likes the philosophy of blocking io too
	default:
		time.Sleep(1 * time.Millisecond)
		f.Push(0)
	}
	f.Next()
}

/*
CODE  TX!   ( c -- )                \ Send character c to output device.
      POP   DX                      \ char in DL
      CMP   DL,0FFH                 \ 0FFH is interpreted as input
      JNZ   TX1                     \ do NOT allow input
      MOV   DL,32                   \ change to blank
TX1:  MOV   AH,6                    \ MS-DOS Direct Console I/O
      INT   021H                    \ display character
      $Next
*/
// ( c -- ) send the character on the data stack
func (f *Forth) _B_TX() {
	out := f.Output
	c := f.Pop()
	fmt.Fprintf(out, "%c", rune(c))
	//fmt.Println("\nTX:", c, fmt.Sprintf("%c", rune(c)))
	/*err := out.Flush()
	if err != nil {
		fmt.Println(err)
	}*/
	f.Next()
}

/*
CODE  doLIT ( -- w )                \ Push inline literal on data stack.
      LODSW                         \ get the literal compiled in-line
      PUSH  AX                      \ push literal on the stack
      $Next                         \ execute next word after literal
*/
// for putting integer literals on the stack
// for data not code
func (f *Forth) doLIT() {
	ax := f.WordPtr(f.IP)
	f.IP += 2
	f.Push(ax)
	f.Next()
}

/*
doLIST is the converse of EXIT.  It pushes the return stack
address onto the data stack.

It's called with CALL doLIST and CALL is a 8086 hack to get
the address of the first word following doLIST on the stack:
the 8086 thinks that is the next instruction.  What a hack?

doLIST      ( a -- )                \ Run address list in a colon word.
      _XCHG  BP,SP                   \ exchange pointers
      PUSH  SI                      \ push return stack
      _XCHG  BP,SP                   \ restore the pointers
      POP   SI                      \ new list address
      $Next
*/
func (f *Forth) doLIST() {
	_XCHG(&f.RP, &f.SP)
	f.Push(f.IP)
	_XCHG(&f.RP, &f.SP)
	f.IP = f.Pop()
	f.Next()
}

/*
   _Call puts the addresss of the first word after doLIST on the stack
   and then then calls the primitive code for the word following itself

   CALL ADDR  ; for example
*/
func (f *Forth) _Call() {
	f.Push(f.WP + 4)
	f.WP = f.WordPtr(f.WP + 2) // move WP over two and down one to the address of doLIST
}

// the converse of doLIST. Ends the colon definition.
/*
CODE  EXIT                          \ Terminate a colon definition.
      _XCHG  BP,SP                   \ exchange pointers
      POP   SI                      \ pop return stack
      _XCHG  BP,SP                   \ restore the pointers
      $Next
*/
func (f *Forth) _EXIT() {
	_XCHG(&f.RP, &f.SP)
	f.IP = f.Pop()
	_XCHG(&f.RP, &f.SP)
	f.Next()
}

/*
CODE  next  ( -- )                  \ Decrement index and exit loop
                                    \ if index is less than 0.
      SUB   WORD PTR [BP],1         \ decrement the index
      JC    Next1                   \ ?decrement below 0
      MOV   SI,0[SI]                \ no, continue loop
      $Next
Next1:ADD   BP,2                    \ yes, pop the index
      ADD   SI,2                    \ exit loop
      $Next
*/
func (f *Forth) _Next() {
	v := asint16(f.WordPtr(f.RP))
	v = v - 1
	f.SetWordPtr(f.RP, asuint16(v))
	//fmt.Printf("prim: _Next() *f.RP is %x\n", f.WordPtr(f.RP))
	if v >= 0 {
		f.IP = f.WordPtr(f.IP)
		//fmt.Printf("%x >= 0 so IP = *IP = %x\n", v, f.IP)
	} else {
		//fmt.Println(v, "< 0 so IP += 2 and RP += 2 ")
		f.RP = f.RP + CELLL
		f.IP = f.IP + CELLL
		//fmt.Printf("RP, IP is %x, %x\n", f.RP, f.IP)
	}
	f.Next()
}

/*
CODE  ?branch     ( f -- )          \ _Branch if flag is zero.
      POP   BX                      \ pop flag
      OR    BX,BX                   \ ?flag=0
      JZ    BRAN1                   \ yes, so branch
      ADD   SI,2                    \ point IP to next cell
      $Next
BRAN1:MOV   SI,0[SI]                \ IP:=(IP), jump to new address
      $Next
*/
func (f *Forth) _Q_branch() {
	bx := f.Pop()
	if bx == 0 {
		f.IP = f.WordPtr(f.IP)
	} else {
		f.IP = f.IP + 2
	}
	f.Next()
}

/*
CODE  branch      ( -- )            \ _Branch to an inline address.
      MOV   SI,0[SI]                \ jump to new address unconditionally
      $Next
*/
func (f *Forth) _Branch() {
	f.IP = f.WordPtr(f.IP)
	f.Next()
}

/*
CODE  !     ( w a -- )              \ Pop the data stack to memory.
      POP   BX                      \ get address from tos
      POP   0[BX]                   \ store data to that adddress
      $Next
*/
func (f *Forth) _Bang() {
	a := f.Pop()
	v := f.Pop()
	f.SetWordPtr(a, v)
	f.Next()
}

/*
CODE  @     ( a -- w )              \ Push memory location to data stack.
      POP   BX                      \ get address
      PUSH  0[BX]                   \ fetch data
      $Next
*/
func (f *Forth) _At() {
	bx := f.Pop()
	v := f.WordPtr(bx)
	f.Push(v)
	f.Next()
}

/*
CODE  C!    ( c b -- )              \ Pop data stack to byte memory.
      POP   BX                      \ get address
      POP   AX                      \ get data in a cell
      MOV   0[BX],AL                \ store one byte
      $Next
*/
func (f *Forth) _Cbang() {
	bx := f.Pop()
	ax := f.Pop()
	f.Memory[bx] = f.RegLower(ax)
	f.Next()
}

/*
CODE  C@    ( b -- c )              \ Push byte memory content on data stack.
      POP   BX                      \ get address
      XOR   AX,AX                   \ AX=0 zero the hi byte
      MOV   AL,0[BX]                \ get low byte
      PUSH  AX                      \ push on stack
      $Next
*/
func (f *Forth) _Cat() {
	bx := f.Pop()
	ax := f.WordPtr(bx)
	ax = 0x00ff & ax
	f.Push(ax)
	f.Next()
}

/*
CODE  RP@   ( -- a )                \ Push current RP to data stack.
      PUSH  BP                      \ copy address to return stack
      $Next                         \ pointer register BP
*/
func (f *Forth) _RPat() {
	f.Push(f.RP)
	f.Next()
}

/*
CODE  RP!   ( a -- )                \ Set the return stack pointer.
      POP   BP                      \ copy (BP) to tos
      $Next
*/
func (f *Forth) _RPbang() {
	f.RP = f.Pop()
	f.Next()
}

/*
CODE  R>    ( -- w )                \ Pop return stack to data stack.
      PUSH  0[BP]                   \ copy w to data stack
      ADD   BP,2                    \ adjust RP for popping
      $Next
*/
func (f *Forth) _Rfrom() {
	f.Push(f.WordPtr(f.RP))
	f.RP = f.RP + 2
	f.Next()
}

/*
CODE  R@    ( -- w )                \ Copy top of return stack to data stack.
      PUSH  0[BP]                   \ copy w to data stack
      $Next
*/
func (f *Forth) _Rat() {
	f.Push(f.WordPtr(f.RP))
	f.Next()
}

/*
CODE  >R    ( w -- )                \ Push data stack to return stack.
      SUB   BP,2                    \ adjust RP for pushing
      POP   0[BP]                   \ push w to return stack
      $Next
*/
func (f *Forth) _Tor() {
	f.RP = f.RP - 2
	f.SetWordPtr(f.RP, f.Pop())
	f.Next()
}

/*
CODE  DROP  ( w -- )                \ Discard top stack item.
      ADD   SP,2                   \ adjust SP to pop
      $Next
*/
func (f *Forth) _Drop() {
	f.SP = f.SP + 2
	f.Next()
}

/*
CODE  DUP   ( w -- w w )            \ _Duplicate the top stack item.
      MOV   BX,SP                   \ use BX to index the stack
      PUSH  0[BX]
      $Next
*/
func (f *Forth) _Dup() {
	f.Push(f.WordPtr(f.SP))
	f.Next()
}

/*
CODE  SWAP  ( w1 w2 -- w2 w1 )      \ Exchange top two stack items.
      POP   BX                      \ get w2
      POP   AX                      \ get w1
      PUSH  BX                      \ push w2
      PUSH  AX                      \ push w1
      $Next
*/
func (f *Forth) _Swap() {
	bx := f.Pop()
	ax := f.Pop()
	f.Push(bx)
	f.Push(ax)
	f.Next()
}

/*
CODE  OVER  ( w1 w2 -- w1 w2 w1 )   \ Copy second stack item to top.
      MOV   BX,SP                   \ use BX to index the stack
      PUSH  2[BX]                   \ get w1 and push on stack
      $Next
*/
func (f *Forth) _Over() {
	bx := f.SP + 2
	f.Push(f.WordPtr(bx))
	f.Next()
}

/*
CODE  SP@   ( -- a )                \ Push the current data stack pointer.
      MOV   BX,SP                   \ use BX to index the stack
      PUSH  BX                      \ push SP back
      $Next
*/
func (f *Forth) _Sp_at() {
	bx := f.SP
	f.Push(bx)
	f.Next()
}

/*
CODE  SP!   ( a -- )                \ Set the data stack pointer.
      POP   SP                      \ safety
      $Next
*/
func (f *Forth) _Sp_bang() {
	f.SP = f.Pop()
	f.Next()
}

/*
CODE  0<    ( n -- f )              \ Return true if n is negative.
      POP   AX
      CWD                           \ sign extend AX into DX
      PUSH  DX                      \ push 0 or -1
      $Next
*/
func (f *Forth) _Zless() {
	ax := asint16(f.Pop())
	if ax >= 0 {
		f.Push(0)
	} else {
		f.Push(asuint16(-1))
	}
	f.Next()
}

/*
CODE  AND   ( w w -- w )            \ Bitwise AND.
      POP   BX
      POP   AX
      AND   BX,AX
      PUSH  BX
      $Next
*/
func (f *Forth) _And() {
	a := f.Pop()
	b := f.Pop()
	r := a & b
	f.Push(r)
	f.Next()
}

/*
CODE  OR    ( w w -- w )            \ Bitwise inclusive OR.
      POP   BX
      POP   AX
      OR    BX,AX
      PUSH  BX
      $Next
*/
func (f *Forth) _Or() {
	a := f.Pop()
	b := f.Pop()
	r := a | b
	f.Push(r)
	f.Next()
}

/*
CODE  XOR   ( w w -- w )            \ Bitwise exclusive OR.
      POP   BX
      POP   AX
      XOR   BX,AX
      PUSH  BX
      $Next
*/
func (f *Forth) _Xor() {
	a := f.Pop()
	b := f.Pop()
	r := a ^ b
	f.Push(r)
	f.Next()
}

/*
CODE  UM+   ( w w -- w cy )
\     Add two numbers, return the sum and carry flag.
      XOR   CX,CX                   \ CX=0 initial carry flag
      POP   BX
      POP   AX
      ADD   AX,BX
      RCL   CX,1                    \ get carry
      PUSH  AX                      \ push sum
      PUSH  CX                      \ push carry
      $Next
*/
func (f *Forth) _UMplus() {
	b := f.Pop()
	a := f.Pop()
	r := a + b
	var cf uint16
	cf = 0
	r32 := uint32(a) + uint32(b)
	if r32 > uint32(r) {
		cf = 1
	}
	f.Push(r)
	f.Push(cf)
	f.Next()
}
