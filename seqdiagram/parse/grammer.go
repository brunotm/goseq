//line grammer.y:6
package parse

import __yyfmt__ "fmt"

//line grammer.y:6
import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/scanner"
)

var DualRunes = map[string]int{
	".": DOT,
	",": COMMA,

	"--": DOUBLEDASH,
	"-":  DASH,
	"=":  EQUAL,

	">>":  DOUBLEANGR,
	">":   ANGR,
	"/>":  SLASHANGR,
	"\\>": BACKSLASHANGR,
}

//line grammer.y:34
type yySymType struct {
	yys          int
	nodeList     *NodeList
	node         Node
	arrow        ArrowType
	arrowStem    ArrowStemType
	arrowHead    ArrowHeadType
	actorRef     ActorRef
	noteAlign    NoteAlignment
	dividerType  GapType
	blockSegList *BlockSegmentList
	attrList     *AttributeList
	attr         *Attribute

	sval string
}

const K_TITLE = 57346
const K_PARTICIPANT = 57347
const K_NOTE = 57348
const K_LEFT = 57349
const K_RIGHT = 57350
const K_OVER = 57351
const K_OF = 57352
const K_HORIZONTAL = 57353
const K_SPACER = 57354
const K_GAP = 57355
const K_LINE = 57356
const K_FRAME = 57357
const K_ALT = 57358
const K_ELSEALT = 57359
const K_ELSE = 57360
const K_END = 57361
const DASH = 57362
const DOUBLEDASH = 57363
const DOT = 57364
const EQUAL = 57365
const COMMA = 57366
const ANGR = 57367
const DOUBLEANGR = 57368
const BACKSLASHANGR = 57369
const SLASHANGR = 57370
const SQRL = 57371
const SQRR = 57372
const MESSAGE = 57373
const IDENT = 57374

var yyToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"K_TITLE",
	"K_PARTICIPANT",
	"K_NOTE",
	"K_LEFT",
	"K_RIGHT",
	"K_OVER",
	"K_OF",
	"K_HORIZONTAL",
	"K_SPACER",
	"K_GAP",
	"K_LINE",
	"K_FRAME",
	"K_ALT",
	"K_ELSEALT",
	"K_ELSE",
	"K_END",
	"DASH",
	"DOUBLEDASH",
	"DOT",
	"EQUAL",
	"COMMA",
	"ANGR",
	"DOUBLEANGR",
	"BACKSLASHANGR",
	"SLASHANGR",
	"SQRL",
	"SQRR",
	"MESSAGE",
	"IDENT",
}
var yyStatenames = [...]string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line grammer.y:253

// Manages the lexer as well as the current diagram being parsed
type parseState struct {
	S     scanner.Scanner
	err   error
	atEof bool
	//diagram     *Diagram
	procInstrs []string
	nodeList   *NodeList
}

func newParseState(src io.Reader, filename string) *parseState {
	ps := &parseState{}
	ps.S.Init(src)
	ps.S.Position.Filename = filename
	//    ps.diagram = &Diagram{}

	return ps
}

func (ps *parseState) Lex(lval *yySymType) int {
	if ps.atEof {
		return 0
	}
	for {
		tok := ps.S.Scan()
		switch tok {
		case scanner.EOF:
			ps.atEof = true
			return 0
		case '#':
			ps.scanComment()
		case ':':
			return ps.scanMessage(lval)
		case '[':
			return SQRL
		case ']':
			return SQRR
		case '-', '>', '*', '=', '/', '\\', '.', ',':
			if res, isTok := ps.handleDoubleRune(tok); isTok {
				return res
			} else {
				ps.Error("Invalid token: " + scanner.TokenString(tok))
			}
		case scanner.Ident:
			return ps.scanKeywordOrIdent(lval)
		default:
			ps.Error("Invalid token: " + scanner.TokenString(tok))
		}
	}
}

func (ps *parseState) handleDoubleRune(firstRune rune) (int, bool) {
	nextRune := ps.S.Peek()

	// Try the double rune
	if nextRune != scanner.EOF {
		tokStr := string(firstRune) + string(nextRune)
		if tok, hasTok := DualRunes[tokStr]; hasTok {
			ps.NextRune()
			return tok, true
		}
	}

	// Try the single rune
	tokStr := string(firstRune)
	if tok, hasTok := DualRunes[tokStr]; hasTok {
		return tok, true
	}

	return 0, false
}

func (ps *parseState) scanKeywordOrIdent(lval *yySymType) int {
	tokVal := ps.S.TokenText()
	switch strings.ToLower(tokVal) {
	case "title":
		return K_TITLE
	case "participant":
		return K_PARTICIPANT
	case "note":
		return K_NOTE
	case "left":
		return K_LEFT
	case "right":
		return K_RIGHT
	case "over":
		return K_OVER
	case "of":
		return K_OF
	case "spacer":
		return K_SPACER
	case "gap":
		return K_GAP
	case "frame":
		return K_FRAME
	case "line":
		return K_LINE
	case "horizontal":
		return K_HORIZONTAL
	case "alt":
		return K_ALT
	case "elsealt":
		return K_ELSEALT
	case "else":
		return K_ELSE
	case "end":
		return K_END
	default:
		lval.sval = tokVal
		return IDENT
	}
}

// Scans a message.  A message is all characters up to the new line
func (ps *parseState) scanMessage(lval *yySymType) int {
	buf := new(bytes.Buffer)
	r := ps.NextRune()
	for (r != '\n') && (r != scanner.EOF) {
		if r == '\\' {
			nr := ps.NextRune()
			switch nr {
			case 'n':
				buf.WriteRune('\n')
			case '\\':
				buf.WriteRune('\\')
			default:
				ps.Error("Invalid backslash escape: \\" + string(nr))
			}
		} else {
			buf.WriteRune(r)
		}
		r = ps.NextRune()
	}

	lval.sval = strings.TrimSpace(buf.String())
	return MESSAGE
}

// Scans a comment.  This ignores all characters up to the new line.
func (ps *parseState) scanComment() {
	var buf *bytes.Buffer

	r := ps.NextRune()
	if r == '!' {
		// This starts a processor instruction
		buf = new(bytes.Buffer)
		r = ps.NextRune()
	}

	for (r != '\n') && (r != scanner.EOF) {
		if buf != nil {
			buf.WriteRune(r)
		}
		r = ps.NextRune()
	}

	if buf != nil {
		ps.procInstrs = append(ps.procInstrs, strings.TrimSpace(buf.String()))
	}
}

func (ps *parseState) NextRune() rune {
	if ps.atEof {
		return scanner.EOF
	}

	r := ps.S.Next()
	if r == scanner.EOF {
		ps.atEof = true
	}

	return r
}

func (ps *parseState) Error(err string) {
	errMsg := fmt.Sprintf("%s:%d: %s", ps.S.Position.Filename, ps.S.Position.Line, err)
	ps.err = errors.New(errMsg)
}

func Parse(reader io.Reader, filename string) (*NodeList, error) {
	ps := newParseState(reader, filename)
	yyParse(ps)

	// Add processing instructions to the start of the node list
	for i := len(ps.procInstrs) - 1; i >= 0; i-- {
		instrParts := strings.SplitN(ps.procInstrs[i], " ", 2)
		name, value := strings.TrimSpace(instrParts[0]), strings.TrimSpace(instrParts[1])
		ps.nodeList = &NodeList{&ProcessInstructionNode{name, value}, ps.nodeList}
	}

	if ps.err != nil {
		return nil, ps.err
	} else {
		return ps.nodeList, nil
	}
}

//line yacctab:1
var yyExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyNprod = 46
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 75

var yyAct = [...]int{

	57, 2, 51, 12, 68, 19, 10, 11, 13, 17,
	18, 17, 18, 14, 53, 56, 21, 69, 15, 66,
	65, 54, 55, 50, 48, 36, 39, 20, 60, 38,
	61, 45, 24, 25, 16, 26, 16, 64, 49, 41,
	42, 43, 44, 62, 59, 58, 32, 33, 34, 35,
	28, 29, 30, 52, 47, 46, 37, 31, 27, 40,
	63, 23, 22, 9, 67, 8, 7, 70, 71, 6,
	5, 4, 72, 3, 1,
}
var yyPact = [...]int{

	2, -1000, -1000, 2, -1000, -1000, -1000, -1000, -1000, -1000,
	-4, -16, 12, 43, 34, -6, -1000, -1000, -1000, -1000,
	-1000, 0, 4, 14, -1000, -1000, -1000, 4, 45, 44,
	-1000, -7, -1000, -1000, -1000, -1000, 2, -8, -18, -10,
	-1000, -1000, -1000, -1000, -1000, -9, -1000, -1000, -1000, 27,
	-1000, -2, 6, 20, -1000, -1000, 4, 18, -11, -12,
	-1000, -18, -28, -14, -1000, 2, 2, -1000, -1000, -1000,
	-1000, 27, -1000,
}
var yyPgo = [...]int{

	0, 74, 1, 73, 71, 70, 69, 66, 65, 63,
	62, 3, 61, 59, 58, 57, 0, 56, 2, 53,
}
var yyR1 = [...]int{

	0, 1, 2, 2, 3, 3, 3, 3, 3, 3,
	4, 17, 17, 18, 18, 18, 19, 5, 5, 6,
	7, 7, 11, 11, 11, 8, 8, 9, 16, 16,
	16, 15, 15, 15, 15, 14, 14, 14, 10, 12,
	12, 12, 13, 13, 13, 13,
}
var yyR2 = [...]int{

	0, 1, 0, 2, 1, 1, 1, 1, 1, 1,
	2, 0, 3, 0, 1, 3, 3, 3, 4, 4,
	4, 6, 1, 1, 1, 2, 3, 5, 0, 3,
	4, 1, 1, 1, 1, 2, 2, 1, 2, 1,
	1, 1, 1, 1, 1, 1,
}
var yyChk = [...]int{

	-1000, -1, -2, -3, -4, -5, -6, -7, -8, -9,
	4, 5, -11, 6, 11, 16, 32, 7, 8, -2,
	31, 32, -10, -12, 20, 21, 23, -14, 7, 8,
	9, -15, 12, 13, 14, 15, 31, -17, 29, -11,
	-13, 25, 26, 27, 28, -11, 10, 10, 31, -2,
	31, -18, -19, 32, 31, 31, 24, -16, 18, 17,
	30, 24, 23, -11, 19, 31, 31, -18, 32, 31,
	-2, -2, -16,
}
var yyDef = [...]int{

	2, -2, 1, 2, 4, 5, 6, 7, 8, 9,
	0, 0, 0, 0, 0, 0, 22, 23, 24, 3,
	10, 11, 0, 0, 39, 40, 41, 0, 0, 0,
	37, 25, 31, 32, 33, 34, 2, 17, 13, 0,
	38, 42, 43, 44, 45, 0, 35, 36, 26, 28,
	18, 0, 14, 0, 19, 20, 0, 0, 0, 0,
	12, 13, 0, 0, 27, 2, 2, 15, 16, 21,
	29, 28, 30,
}
var yyTok1 = [...]int{

	1,
}
var yyTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32,
}
var yyTok3 = [...]int{
	0,
}

var yyErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	yyDebug        = 0
	yyErrorVerbose = false
)

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyParser interface {
	Parse(yyLexer) int
	Lookahead() int
}

type yyParserImpl struct {
	lookahead func() int
}

func (p *yyParserImpl) Lookahead() int {
	return p.lookahead()
}

func yyNewParser() yyParser {
	p := &yyParserImpl{
		lookahead: func() int { return -1 },
	}
	return p
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c >= 1 && c-1 < len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yyErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !yyErrorVerbose {
		return "syntax error"
	}

	for _, e := range yyErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + yyTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := yyPact[state]
	for tok := TOKSTART; tok-1 < len(yyToknames); tok++ {
		if n := base + tok; n >= 0 && n < yyLast && yyChk[yyAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if yyDef[state] == -2 {
		i := 0
		for yyExca[i] != -1 || yyExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; yyExca[i] >= 0; i += 2 {
			tok := yyExca[i]
			if tok < TOKSTART || yyExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if yyExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += yyTokname(tok)
	}
	return res
}

func yylex1(lex yyLexer, lval *yySymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		token = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			token = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		token = yyTok3[i+0]
		if token == char {
			token = yyTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(token), uint(char))
	}
	return char, token
}

func yyParse(yylex yyLexer) int {
	return yyNewParser().Parse(yylex)
}

func (yyrcvr *yyParserImpl) Parse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	var yyDollar []yySymType
	_ = yyDollar // silence set and not used
	yyS := make([]yySymType, yyMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yytoken := -1 // yychar translated into internal numbering
	yyrcvr.lookahead = func() int { return yychar }
	defer func() {
		// Make sure we report no lookahead when not parsing.
		yystate = -1
		yychar = -1
		yytoken = -1
	}()
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yytoken), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yychar < 0 {
		yychar, yytoken = yylex1(yylex, &yylval)
	}
	yyn += yytoken
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yytoken { /* valid shift */
		yychar = -1
		yytoken = -1
		yyVAL = yylval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yychar < 0 {
			yychar, yytoken = yylex1(yylex, &yylval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yytoken {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error(yyErrorMessage(yystate, yytoken))
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yytoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yytoken))
			}
			if yytoken == yyEofCode {
				goto ret1
			}
			yychar = -1
			yytoken = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	// yyp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:79
		{
			yylex.(*parseState).nodeList = yyDollar[1].nodeList
		}
	case 2:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line grammer.y:86
		{
			yyVAL.nodeList = nil
		}
	case 3:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammer.y:90
		{
			yyVAL.nodeList = &NodeList{yyDollar[1].node, yyDollar[2].nodeList}
		}
	case 10:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammer.y:106
		{
			yyVAL.node = &TitleNode{yyDollar[2].sval}
		}
	case 11:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line grammer.y:113
		{
			yyVAL.attrList = nil
		}
	case 12:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammer.y:117
		{
			yyVAL.attrList = yyDollar[2].attrList
		}
	case 13:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line grammer.y:124
		{
			yyVAL.attrList = nil
		}
	case 14:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:128
		{
			yyVAL.attrList = &AttributeList{yyDollar[1].attr, nil}
		}
	case 15:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammer.y:132
		{
			yyVAL.attrList = &AttributeList{yyDollar[1].attr, yyDollar[3].attrList}
		}
	case 16:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammer.y:139
		{
			yyVAL.attr = &Attribute{yyDollar[1].sval, yyDollar[3].sval}
		}
	case 17:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammer.y:146
		{
			yyVAL.node = &ActorNode{yyDollar[2].sval, false, "", yyDollar[3].attrList}
		}
	case 18:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line grammer.y:150
		{
			yyVAL.node = &ActorNode{yyDollar[2].sval, true, yyDollar[4].sval, yyDollar[3].attrList}
		}
	case 19:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line grammer.y:157
		{
			yyVAL.node = &ActionNode{yyDollar[1].actorRef, yyDollar[3].actorRef, yyDollar[2].arrow, yyDollar[4].sval}
		}
	case 20:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line grammer.y:164
		{
			yyVAL.node = &NoteNode{yyDollar[3].actorRef, nil, yyDollar[2].noteAlign, yyDollar[4].sval}
		}
	case 21:
		yyDollar = yyS[yypt-6 : yypt+1]
		//line grammer.y:168
		{
			yyVAL.node = &NoteNode{yyDollar[3].actorRef, yyDollar[5].actorRef, yyDollar[2].noteAlign, yyDollar[6].sval}
		}
	case 22:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:175
		{
			yyVAL.actorRef = NormalActorRef(yyDollar[1].sval)
		}
	case 23:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:179
		{
			yyVAL.actorRef = PseudoActorRef("left")
		}
	case 24:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:183
		{
			yyVAL.actorRef = PseudoActorRef("right")
		}
	case 25:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammer.y:190
		{
			yyVAL.node = &GapNode{yyDollar[2].dividerType, ""}
		}
	case 26:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammer.y:194
		{
			yyVAL.node = &GapNode{yyDollar[2].dividerType, yyDollar[3].sval}
		}
	case 27:
		yyDollar = yyS[yypt-5 : yypt+1]
		//line grammer.y:201
		{
			yyVAL.node = &BlockNode{&BlockSegmentList{&BlockSegment{ALT_SEGMENT, "", yyDollar[2].sval, yyDollar[3].nodeList}, yyDollar[4].blockSegList}}
		}
	case 28:
		yyDollar = yyS[yypt-0 : yypt+1]
		//line grammer.y:208
		{
			yyVAL.blockSegList = nil
		}
	case 29:
		yyDollar = yyS[yypt-3 : yypt+1]
		//line grammer.y:212
		{
			yyVAL.blockSegList = &BlockSegmentList{&BlockSegment{ALT_ELSE_SEGMENT, "", yyDollar[2].sval, yyDollar[3].nodeList}, nil}
		}
	case 30:
		yyDollar = yyS[yypt-4 : yypt+1]
		//line grammer.y:216
		{
			yyVAL.blockSegList = &BlockSegmentList{&BlockSegment{ALT_SEGMENT, "", yyDollar[2].sval, yyDollar[3].nodeList}, yyDollar[4].blockSegList}
		}
	case 31:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:222
		{
			yyVAL.dividerType = SPACER_GAP
		}
	case 32:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:223
		{
			yyVAL.dividerType = EMPTY_GAP
		}
	case 33:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:224
		{
			yyVAL.dividerType = LINE_GAP
		}
	case 34:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:225
		{
			yyVAL.dividerType = FRAME_GAP
		}
	case 35:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammer.y:229
		{
			yyVAL.noteAlign = LEFT_NOTE_ALIGNMENT
		}
	case 36:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammer.y:230
		{
			yyVAL.noteAlign = RIGHT_NOTE_ALIGNMENT
		}
	case 37:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:231
		{
			yyVAL.noteAlign = OVER_NOTE_ALIGNMENT
		}
	case 38:
		yyDollar = yyS[yypt-2 : yypt+1]
		//line grammer.y:236
		{
			yyVAL.arrow = ArrowType{yyDollar[1].arrowStem, yyDollar[2].arrowHead}
		}
	case 39:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:242
		{
			yyVAL.arrowStem = SOLID_ARROW_STEM
		}
	case 40:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:243
		{
			yyVAL.arrowStem = DASHED_ARROW_STEM
		}
	case 41:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:244
		{
			yyVAL.arrowStem = THICK_ARROW_STEM
		}
	case 42:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:248
		{
			yyVAL.arrowHead = SOLID_ARROW_HEAD
		}
	case 43:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:249
		{
			yyVAL.arrowHead = OPEN_ARROW_HEAD
		}
	case 44:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:250
		{
			yyVAL.arrowHead = BARBED_ARROW_HEAD
		}
	case 45:
		yyDollar = yyS[yypt-1 : yypt+1]
		//line grammer.y:251
		{
			yyVAL.arrowHead = LOWER_BARBED_ARROW_HEAD
		}
	}
	goto yystack /* stack new state and value */
}
