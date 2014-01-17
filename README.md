# eforth #

This is an implementation of eforth in golang.  Some have said it ought to be called go forth, but they don't know go eschews such attention, being a boring behind the scenes systems language who prefers soup to leg of lamb with appricot glaze, and not some primadona like JavaScript.

## Try it ##

On *nix just clone the repo and then in the eforth directory do

    cd example; go build
    ./forth 

    eForth 0.01
    WORDS
    WORDS
    COLD 'BOOT hi VER WORDS SEE .ID >NAME ?CSP !CSP .S DUMP dm+ _TYPE VARIABLE CREATE USER IMMEDIATE : call, ] ; OVERT $COMPILE $,n ?UNIQUE ." $" ABORT" WHILE ELSE AFT THEN REPEAT AHEAD IF AGAIN UNTIL NEXT BEGIN FOR RECURSE $," LITERAL COMPILE [COMPILE] , ALLOT ' QUIT CONSOLE I/O HAND FILE xio PRESET EVAL ?STACK .OK [ $INTERPRET abort" ABORT NULL$ THROW CATCH QUERY EXPECT accept kTAP TAP ^H NAME? find SAME? NAME> WORD TOKEN CHAR \ ( .( PARSE parse ? . U. U.R .R ."| $"| do$ CR TYPE SPACES SPACE PACE NUF? EMIT KEY ?KEY NUMBER? DIGIT? DECIMAL HEX str #> SIGN #S # HOLD <# EXTRACT DIGIT PACK$ -TRAILING FILL CMOVE @EXECUTE TIB PAD HERE COUNT 2@ 2! +! PICK DEPTH >CHAR BL ALIGNED CELLS CELL- CELL+ */ */MOD M* * UM* / MOD /MOD M/MOD UM/MOD WITHIN MIN MAX < U< = ABS - DNEGATE NEGATE NOT D+ + 2DUP 2DROP ROT ?DUP FORTH doVOC LAST NP CP CURRENT CONTEXT HANDLER HLD 'NUMBER 'EVAL CSP #TIB >IN SPAN tmp BASE 'PROMPT 'ECHO 'TAP 'EXPECT 'EMIT '?KEY RP0 SP0 doUSER + ROT UP doVAR UM+ XOR OR AND 0< SP! SP@ OVER SWAP DUP DROP >R R@ R> RP! RP@ C@ C! @ ! branch ?branch next EXIT doLIT EXECUTE TX! ?RX !IO doLIST CALL BYE ok
