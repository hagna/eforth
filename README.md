# What is it for? #

Say you have a computer game, and you want to let users script enemy behavior.  You know you want to embed some programming language into your game, but you haven't done it yet because you are still reading something like this http://www.goodmath.org/blog/2014/05/04/combinator-parsing-part-1/ or reading about how to restrict lua after it is embedded or maybe you just have the idea, but don't have the drive to make it happen yet.  This eforth aims to deliver just enough programming language with comparitively little embedding effort.

# What is it? #

This is EFORTH simulated in golang, and EFORTH is just a variant of FORTH (http://galileo.phys.virginia.edu/classes/551.jvn.fall01/primer.htm)



## Try it ##

To try it do this:

    go get github.com/hagna/eforth
    go install github.com/hagna/eforth/eforth_repl
    
    $GOPATH/bin/eforth_repl

    eForth 0.01
    WORDS
    WORDS
    COLD 'BOOT hi VER WORDS SEE .ID >NAME ?CSP !CSP .S DUMP dm+ _TYPE VARIABLE CREATE USER IMMEDIATE : call, ] ; OVERT $COMPILE $,n ?UNIQUE ." $" ABORT" WHILE ELSE AFT THEN REPEAT AHEAD IF AGAIN UNTIL NEXT BEGIN FOR RECURSE $," LITERAL COMPILE [COMPILE] , ALLOT ' QUIT CONSOLE I/O HAND FILE xio PRESET EVAL ?STACK .OK [ $INTERPRET abort" ABORT NULL$ THROW CATCH QUERY EXPECT accept kTAP TAP ^H NAME? find SAME? NAME> WORD TOKEN CHAR \ ( .( PARSE parse ? . U. U.R .R ."| $"| do$ CR TYPE SPACES SPACE PACE NUF? EMIT KEY ?KEY NUMBER? DIGIT? DECIMAL HEX str #> SIGN #S # HOLD <# EXTRACT DIGIT PACK$ -TRAILING FILL CMOVE @EXECUTE TIB PAD HERE COUNT 2@ 2! +! PICK DEPTH >CHAR BL ALIGNED CELLS CELL- CELL+ */ */MOD M* * UM* / MOD /MOD M/MOD UM/MOD WITHIN MIN MAX < U< = ABS - DNEGATE NEGATE NOT D+ + 2DUP 2DROP ROT ?DUP FORTH doVOC LAST NP CP CURRENT CONTEXT HANDLER HLD 'NUMBER 'EVAL CSP #TIB >IN SPAN tmp BASE 'PROMPT 'ECHO 'TAP 'EXPECT 'EMIT '?KEY RP0 SP0 doUSER + ROT UP doVAR UM+ XOR OR AND 0< SP! SP@ OVER SWAP DUP DROP >R R@ R> RP! RP@ C@ C! @ ! branch ?branch next EXIT doLIT EXECUTE TX! ?RX !IO doLIST CALL BYE ok
