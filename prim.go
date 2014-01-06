package eforth

import (
	"bufio"
	"fmt"
	"os"
)

func (f *Forth) AddPrimitives() {
	words := []struct {
		word string
		m    fn
	}{
		{"BYE", f.BYE},
		{"CALL", f.Call},
		{"doLIST", f.doLIST},
		{"!IO", f.B_IO},
		{"?RX", f.Q_RX},
		{"TX!", f.B_TX},
		{"EXECUTE", f.Execute},
		{"doLIT", f.doLIT},
		{"EXIT", f.EXIT},
		{"next", f.Next},
		{"?branch", f.Q_branch},
		{"branch", f.Branch},
		{"!", f.Bang},
		{"@", f.At},
		{"C!", f.Cbang},
		{"C@", f.Cat},
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
	for _, v := range words {
		f.AddPrim(v.word, v.m)
	}
}

/*
CODE BYE    ( -- , exit Forth )
      INT   020H                    \ return to DOS
*/
func (f *Forth) BYE() {
	f.IP = 0xffff
}

/*
CODE  !IO   ( -- )                  \ Initialize the serial I/O devices.
      $_next
*/
// initialize IO
func (f *Forth) B_IO() {
	f.input = bufio.NewReader(os.Stdin)
	f.output = bufio.NewWriter(os.Stdout)
	f._next()
}

/*
CODE  EXECUTE     ( ca -- )         \ Execute the word at ca.
      POP   BX
      JMP   BX                      \ jump to the code address
*/
func (f *Forth) Execute() {
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
      $_next
*/
// RX may need to be non-blocking receive
// returns either false or char true
func (f *Forth) Q_RX() {
	in := f.input.(*bufio.Reader)
	b, err := in.ReadByte()
	if err != nil {
		fmt.Println("could not read Byte", err)
		f.Push(0)
	} else {
		f.Push(uint16(b))
		f.Push(asuint16(-1))
	}
	f._next()
}

/*
CODE  TX!   ( c -- )                \ Send character c to output device.
      POP   DX                      \ char in DL
      CMP   DL,0FFH                 \ 0FFH is interpreted as input
      JNZ   TX1                     \ do NOT allow input
      MOV   DL,32                   \ change to blank
TX1:  MOV   AH,6                    \ MS-DOS Direct Console I/O
      INT   021H                    \ display character
      $_next
*/
// ( c -- ) send the character on the data stack
func (f *Forth) B_TX() {
	out := f.output.(*bufio.Writer)
	c := f.Pop()
	fmt.Fprintf(out, "%c", rune(c))
	fmt.Println("output:", c, fmt.Sprintf("%c", rune(c)))
	err := out.Flush()
	if err != nil {
		fmt.Println(err)
	}
	f._next()
}

/*
CODE  doLIT ( -- w )                \ Push inline literal on data stack.
      LODSW                         \ get the literal compiled in-line
      PUSH  AX                      \ push literal on the stack
      $_next                         \ execute next word after literal
*/
// for putting integer literals on the stack
// for data not code
func (f *Forth) doLIT() {
	ax := f.WordPtr(f.IP)
	f.IP += 2
	f.Push(ax)
	f._next()
}

/*
doLIST is the converse of EXIT.  It pushes the return stack
address onto the data stack.

It's called with CALL doLIST and CALL is a 8086 hack to get
the address of the first word following doLIST on the stack:
the 8086 thinks that is the next instruction.  What a hack?

doLIST      ( a -- )                \ Run address list in a colon word.
      XCHG  BP,SP                   \ exchange pointers
      PUSH  SI                      \ push return stack
      XCHG  BP,SP                   \ restore the pointers
      POP   SI                      \ new list address
      $_next
*/
func (f *Forth) doLIST() {
	XCHG(&f.RP, &f.SP)
	f.Push(f.IP)
	XCHG(&f.RP, &f.SP)
	f.IP = f.Pop()
	f._next()
}

/*
   Call puts the addresss of the first word after doLIST on the stack
   and then then calls the primitive code for the word following itself

   CALL ADDR  ; for example
*/
func (f *Forth) Call() {
	f.Push(f.WP + 4)
	f.WP = f.WordPtr(f.WP + 2) // move WP over two and down one to the address of doLIST
}

// the converse of doLIST. Ends the colon definition.
/*
CODE  EXIT                          \ Terminate a colon definition.
      XCHG  BP,SP                   \ exchange pointers
      POP   SI                      \ pop return stack
      XCHG  BP,SP                   \ restore the pointers
      $_next
*/
func (f *Forth) EXIT() {
	XCHG(&f.RP, &f.SP)
	f.IP = f.Pop()
	XCHG(&f.RP, &f.SP)
	f._next()
}

/*
CODE  next  ( -- )                  \ Decrement index and exit loop
                                    \ if index is less than 0.
      SUB   WORD PTR [BP],1         \ decrement the index
      JC    _next1                   \ ?decrement below 0
      MOV   SI,0[SI]                \ no, continue loop
      $_next
_next1:ADD   BP,2                    \ yes, pop the index
      ADD   SI,2                    \ exit loop
      $_next
*/
func (f *Forth) Next() {
	v := asint16(f.WordPtr(f.RP))
	v = v - 1
	f.SetWordPtr(f.RP, asuint16(v))
	//fmt.Println("prim: Next() f.RP is", f.WordPtr(f.RP))
	if v >= 0 {
		f.IP = f.WordPtr(f.IP)
		//fmt.Println(">= 0 so IP = *IP")
	} else {
		//fmt.Println("< 0 so IP += 2 and RP += 2 ")
		f.RP = f.RP + 2
		f.IP = f.IP + 2
		//fmt.Println("RP is", f.RP, "and f.IP is", f.IP)
	}
	f._next()
}

/*
CODE  ?branch     ( f -- )          \ Branch if flag is zero.
      POP   BX                      \ pop flag
      OR    BX,BX                   \ ?flag=0
      JZ    BRAN1                   \ yes, so branch
      ADD   SI,2                    \ point IP to next cell
      $_next
BRAN1:MOV   SI,0[SI]                \ IP:=(IP), jump to new address
      $_next
*/
func (f *Forth) Q_branch() {
	bx := f.Pop()
	if bx == 0 {
		f.IP = f.WordPtr(f.IP)
	} else {
		f.IP = f.IP + 2
	}
	f._next()
}

/*
CODE  branch      ( -- )            \ Branch to an inline address.
      MOV   SI,0[SI]                \ jump to new address unconditionally
      $_next
*/
func (f *Forth) Branch() {
	f.IP = f.WordPtr(f.IP)
	f._next()
}

/*
CODE  !     ( w a -- )              \ Pop the data stack to memory.
      POP   BX                      \ get address from tos
      POP   0[BX]                   \ store data to that adddress
      $_next
*/
func (f *Forth) Bang() {
	a := f.Pop()
	v := f.Pop()
	f.SetWordPtr(a, v)
	f._next()
}

/*
CODE  @     ( a -- w )              \ Push memory location to data stack.
      POP   BX                      \ get address
      PUSH  0[BX]                   \ fetch data
      $_next
*/
func (f *Forth) At() {
	bx := f.Pop()
	v := f.WordPtr(bx)
	f.Push(v)
	f._next()
}

/*
CODE  C!    ( c b -- )              \ Pop data stack to byte memory.
      POP   BX                      \ get address
      POP   AX                      \ get data in a cell
      MOV   0[BX],AL                \ store one byte
      $_next
*/
func (f *Forth) Cbang() {
	bx := f.Pop()
	ax := f.Pop()
	f.SetBytePtr(bx, f.RegLower(ax))
	f._next()
}

/*
CODE  C@    ( b -- c )              \ Push byte memory content on data stack.
      POP   BX                      \ get address
      XOR   AX,AX                   \ AX=0 zero the hi byte
      MOV   AL,0[BX]                \ get low byte
      PUSH  AX                      \ push on stack
      $_next
*/
func (f *Forth) Cat() {
	bx := f.Pop()
	ax := f.WordPtr(bx)
	ax = 0x00ff & ax
	f.Push(ax)
	f._next()
}

/*
CODE  RP@   ( -- a )                \ Push current RP to data stack.
      PUSH  BP                      \ copy address to return stack
      $_next                         \ pointer register BP
*/
func (f *Forth) RPat() {
	f.Push(f.RP)
	f._next()
}

/*
CODE  RP!   ( a -- )                \ Set the return stack pointer.
      POP   BP                      \ copy (BP) to tos
      $_next
*/
func (f *Forth) RPbang() {
	f.RP = f.Pop()
	f._next()
}

/*
CODE  R>    ( -- w )                \ Pop return stack to data stack.
      PUSH  0[BP]                   \ copy w to data stack
      ADD   BP,2                    \ adjust RP for popping
      $_next
*/
func (f *Forth) Rfrom() {
	f.Push(f.WordPtr(f.RP))
	f.RP = f.RP + 2
	f._next()
}

/*
CODE  R@    ( -- w )                \ Copy top of return stack to data stack.
      PUSH  0[BP]                   \ copy w to data stack
      $_next
*/
func (f *Forth) Rat() {
	f.Push(f.WordPtr(f.RP))
	f._next()
}

/*
CODE  >R    ( w -- )                \ Push data stack to return stack.
      SUB   BP,2                    \ adjust RP for pushing
      POP   0[BP]                   \ push w to return stack
      $_next
*/
func (f *Forth) Tor() {
	f.RP = f.RP - 2
	f.SetWordPtr(f.RP, f.Pop())
	f._next()
}

/*
CODE  DROP  ( w -- )                \ Discard top stack item.
      ADD   SP,2                   \ adjust SP to pop
      $_next
*/
func (f *Forth) Drop() {
	f.SP = f.SP + 2
	f._next()
}

/*
CODE  DUP   ( w -- w w )            \ Duplicate the top stack item.
      MOV   BX,SP                   \ use BX to index the stack
      PUSH  0[BX]
      $_next
*/
func (f *Forth) Dup() {
	f.Push(f.WordPtr(f.SP))
	f._next()
}

/*
CODE  SWAP  ( w1 w2 -- w2 w1 )      \ Exchange top two stack items.
      POP   BX                      \ get w2
      POP   AX                      \ get w1
      PUSH  BX                      \ push w2
      PUSH  AX                      \ push w1
      $_next
*/
func (f *Forth) Swap() {
	bx := f.Pop()
	ax := f.Pop()
	f.Push(bx)
	f.Push(ax)
	f._next()
}

/*
CODE  OVER  ( w1 w2 -- w1 w2 w1 )   \ Copy second stack item to top.
      MOV   BX,SP                   \ use BX to index the stack
      PUSH  2[BX]                   \ get w1 and push on stack
      $_next
*/
func (f *Forth) Over() {
	bx := f.SP + 2
	f.Push(f.WordPtr(bx))
	f._next()
}

/*
CODE  SP@   ( -- a )                \ Push the current data stack pointer.
      MOV   BX,SP                   \ use BX to index the stack
      PUSH  BX                      \ push SP back
      $_next
*/
func (f *Forth) Sp_at() {
	bx := f.SP
	f.Push(bx)
	f._next()
}

/*
CODE  SP!   ( a -- )                \ Set the data stack pointer.
      POP   SP                      \ safety
      $_next
*/
func (f *Forth) Sp_bang() {
	f.SP = f.Pop()
	f._next()
}

/*
CODE  0<    ( n -- f )              \ Return true if n is negative.
      POP   AX
      CWD                           \ sign extend AX into DX
      PUSH  DX                      \ push 0 or -1
      $_next
*/
func (f *Forth) Zless() {
	ax := asint16(f.Pop())
	if ax >= 0 {
		f.Push(0)
	} else {
		f.Push(asuint16(-1))
	}
	f._next()
}

/*
CODE  AND   ( w w -- w )            \ Bitwise AND.
      POP   BX
      POP   AX
      AND   BX,AX
      PUSH  BX
      $_next
*/
func (f *Forth) And() {
	a := f.Pop()
	b := f.Pop()
	r := a & b
	f.Push(r)
	f._next()
}

/*
CODE  OR    ( w w -- w )            \ Bitwise inclusive OR.
      POP   BX
      POP   AX
      OR    BX,AX
      PUSH  BX
      $_next
*/
func (f *Forth) Or() {
	a := f.Pop()
	b := f.Pop()
	r := a | b
	f.Push(r)
	f._next()
}

/*
CODE  XOR   ( w w -- w )            \ Bitwise exclusive OR.
      POP   BX
      POP   AX
      XOR   BX,AX
      PUSH  BX
      $_next
*/
func (f *Forth) Xor() {
	a := f.Pop()
	b := f.Pop()
	r := a ^ b
	f.Push(r)
	f._next()
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
      $_next
*/
func (f *Forth) UMplus() {
	b := asint16(f.Pop())
	a := asint16(f.Pop())
	r := a + b
	var cf uint16
	cf = 0
	r32 := uint32(a) + uint32(b)
	if r32 > uint32(r) {
		cf = 1
	}
	f.Push(asuint16(r))
	f.Push(cf)
	f._next()
}
