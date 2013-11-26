package eforth

import (
	"bufio"
	"fmt"
	"os"
)

func (f *Forth) BYE() {
	f.WP = 0xffff
}

// initialize IO
func (f *Forth) B_IO() {
	f.input = bufio.NewReader(os.Stdin)
	f.output = bufio.NewWriter(os.Stdout)
}

func (f *Forth) Execute() {
	bx := f.Pop()
	f.IP = bx
}

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
		f.Push(^uint16(0))
	}
}

// ( c -- ) send the character on the data stack
func (f *Forth) B_TX() {
	out := f.output.(*bufio.Writer)
	c := f.Pop()
	fmt.Fprintf(out, "%c", rune(c))
	err := out.Flush()
	if err != nil {
		fmt.Println(err)
	}
}

// for putting integer literals on the stack
// for data not code
func (f *Forth) doLIT() {
	f.NEXT() // shortcut because next is just LODSW in this implementation and not the full
	/* CODE NEXT:
	   LODSW
	   JMP AX
	*/
	f.Push(f.WP)
}

// doLIST is the converse of EXIT.  It pushes the return stack
// address onto the data stack
/*
doLIST      ( a -- )                \ Run address list in a colon word.
      XCHG  BP,SP                   \ exchange pointers
      PUSH  SI                      \ push return stack
      XCHG  BP,SP                   \ restore the pointers
      POP   SI                      \ new list address
      $NEXT
*/
func (f *Forth) doLIST() {
	XCHG(&f.RP, &f.SP)
	f.Push(f.IP)
	XCHG(&f.RP, &f.SP)
	f.IP = f.Pop()
}

// the converse of doLIST. Ends the colon definition.
/*
CODE  EXIT                          \ Terminate a colon definition.
      XCHG  BP,SP                   \ exchange pointers
      POP   SI                      \ pop return stack
      XCHG  BP,SP                   \ restore the pointers
      $NEXT
*/
func (f *Forth) EXIT() {
	XCHG(&f.RP, &f.SP)
	f.IP = f.Pop()
	XCHG(&f.RP, &f.SP)
}

/*
CODE  next  ( -- )                  \ Decrement index and exit loop
                                    \ if index is less than 0.
      SUB   WORD PTR [BP],1         \ decrement the index
      JC    NEXT1                   \ ?decrement below 0
      MOV   SI,0[SI]                \ no, continue loop
      $NEXT
NEXT1:ADD   BP,2                    \ yes, pop the index
      ADD   SI,2                    \ exit loop
      $NEXT
*/
func (f *Forth) Next() {
	v := f.WordPtr(f.RP)
	v = v - 1
	f.SetWordPtr(f.RP, v)
	if v >= 0 {
		f.IP = f.WordPtr(f.IP)
	} else {
		f.RP = f.RP + 2
		f.IP = f.IP + 2
	}
}

/*
CODE  ?branch     ( f -- )          \ Branch if flag is zero.
      POP   BX                      \ pop flag
      OR    BX,BX                   \ ?flag=0
      JZ    BRAN1                   \ yes, so branch
      ADD   SI,2                    \ point IP to next cell
      $NEXT
BRAN1:MOV   SI,0[SI]                \ IP:=(IP), jump to new address
      $NEXT
*/
func (f *Forth) Q_branch() {
	bx := f.Pop()
	if bx == 0 {
		f.IP = f.WordPtr(f.IP)
	} else {
		f.IP = f.IP + 2
	}
}

/*
CODE  branch      ( -- )            \ Branch to an inline address.
      MOV   SI,0[SI]                \ jump to new address unconditionally
      $NEXT
*/
func (f *Forth) Branch() {
	f.IP = f.WordPtr(f.IP)
}

/*
CODE  !     ( w a -- )              \ Pop the data stack to memory.
      POP   BX                      \ get address from tos
      POP   0[BX]                   \ store data to that adddress
      $NEXT
*/
func (f *Forth) Bang() {
	a := f.Pop()
	v := f.Pop()
	f.SetWordPtr(a, v)
}

/*
CODE  @     ( a -- w )              \ Push memory location to data stack.
      POP   BX                      \ get address
      PUSH  0[BX]                   \ fetch data
      $NEXT
*/
func (f *Forth) At() {
	bx := f.Pop()
	v := f.WordPtr(bx)
	f.Push(v)
}

/*
CODE  C!    ( c b -- )              \ Pop data stack to byte memory.
      POP   BX                      \ get address
      POP   AX                      \ get data in a cell
      MOV   0[BX],AL                \ store one byte
      $NEXT
*/
func (f *Forth) Cbang() {
	bx := f.Pop()
	ax := f.Pop()
	f.SetBytePtr(bx, f.RegLower(ax))
}

/*
CODE  C@    ( b -- c )              \ Push byte memory content on data stack.
      POP   BX                      \ get address
      XOR   AX,AX                   \ AX=0 zero the hi byte
      MOV   AL,0[BX]                \ get low byte
      PUSH  AX                      \ push on stack
      $NEXT
*/
func (f *Forth) Cat() {
	bx := f.Pop()
	ax := f.WordPtr(bx)
	ax = 0x00ff & ax
	f.Push(ax)
}

/*
CODE  RP@   ( -- a )                \ Push current RP to data stack.
      PUSH  BP                      \ copy address to return stack
      $NEXT                         \ pointer register BP
*/
func (f *Forth) RPat() {
	f.Push(f.RP)
}

/*
CODE  RP!   ( a -- )                \ Set the return stack pointer.
      POP   BP                      \ copy (BP) to tos
      $NEXT
*/
func (f *Forth) RPbang() {
	f.RP = f.Pop()
}

/*
CODE  R>    ( -- w )                \ Pop return stack to data stack.
      PUSH  0[BP]                   \ copy w to data stack
      ADD   BP,2                    \ adjust RP for popping
      $NEXT
*/
func (f *Forth) Rfrom() {
	f.Push(f.WordPtr(f.RP))
	f.RP = f.RP + 2
}

/*
CODE  R@    ( -- w )                \ Copy top of return stack to data stack.
      PUSH  0[BP]                   \ copy w to data stack
      $NEXT
*/
func (f *Forth) Rat() {
	f.Push(f.WordPtr(f.RP))
}

/*
CODE  >R    ( w -- )                \ Push data stack to return stack.
      SUB   BP,2                    \ adjust RP for pushing
      POP   0[BP]                   \ push w to return stack
      $NEXT
*/
func (f *Forth) Tor() {
	f.RP = f.RP - 2
	f.SetWordPtr(f.RP, f.Pop())
}

/*
CODE  DROP  ( w -- )                \ Discard top stack item.
      ADD   SP,2                   \ adjust SP to pop
      $NEXT
*/
func (f *Forth) Drop() {
	f.SP = f.SP + 2
}

/*
CODE  DUP   ( w -- w w )            \ Duplicate the top stack item.
      MOV   BX,SP                   \ use BX to index the stack
      PUSH  0[BX]
      $NEXT
*/
func (f *Forth) Dup() {
	f.Push(f.WordPtr(f.SP))
}

/*
CODE  SWAP  ( w1 w2 -- w2 w1 )      \ Exchange top two stack items.
      POP   BX                      \ get w2
      POP   AX                      \ get w1
      PUSH  BX                      \ push w2
      PUSH  AX                      \ push w1
      $NEXT
*/
func (f *Forth) Swap() {
	bx := f.Pop()
	ax := f.Pop()
	f.Push(bx)
	f.Push(ax)
}

/*
CODE  OVER  ( w1 w2 -- w1 w2 w1 )   \ Copy second stack item to top.
      MOV   BX,SP                   \ use BX to index the stack
      PUSH  2[BX]                   \ get w1 and push on stack
      $NEXT
*/
func (f *Forth) Over() {
	bx := f.SP + 2
	f.Push(f.WordPtr(bx))
}

/*
CODE  SP@   ( -- a )                \ Push the current data stack pointer.
      MOV   BX,SP                   \ use BX to index the stack
      PUSH  BX                      \ push SP back
      $NEXT
*/
func (f *Forth) Sp_at() {
	bx := f.SP
	f.Push(bx)
}

/*
CODE  SP!   ( a -- )                \ Set the data stack pointer.
      POP   SP                      \ safety
      $NEXT
*/
func (f *Forth) Sp_bang() {
	f.SP = f.Pop()
}

/*
CODE  0<    ( n -- f )              \ Return true if n is negative.
      POP   AX
      CWD                           \ sign extend AX into DX
      PUSH  DX                      \ push 0 or -1
      $NEXT
*/
func (f *Forth) Zless() {
	ax := f.Pop()
	if ax >= 0 {
		f.Push(0)
	} else {
		f.Push(0xffff)
	}
}

/*
CODE  AND   ( w w -- w )            \ Bitwise AND.
      POP   BX
      POP   AX
      AND   BX,AX
      PUSH  BX
      $NEXT
*/
func (f *Forth) And() {
	a := f.Pop()
	b := f.Pop()
	r := a & b
	f.Push(r)
}

/*
CODE  OR    ( w w -- w )            \ Bitwise inclusive OR.
      POP   BX
      POP   AX
      OR    BX,AX
      PUSH  BX
      $NEXT
*/
func (f *Forth) Or() {
	a := f.Pop()
	b := f.Pop()
	r := a | b
	f.Push(r)
}

/*
CODE  XOR   ( w w -- w )            \ Bitwise exclusive OR.
      POP   BX
      POP   AX
      XOR   BX,AX
      PUSH  BX
      $NEXT
*/
func (f *Forth) Xor() {
	a := f.Pop()
	b := f.Pop()
	r := a ^ b
	f.Push(r)
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
      $NEXT
*/
func (f *Forth) UMplus() {
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
}
