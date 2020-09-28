package expression

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DynamoGraph/expression/lexer"
	"github.com/DynamoGraph/expression/token"
)

type (
	Parser struct {
		l *lexer.Lexer

		extend bool

		abort     bool
		stmtType  string
		curToken  *token.Token
		peekToken *token.Token

		parseFns map[token.TokenType]funcs.FuncT //TODO: why include this as part of Parser. Make pkg var instead. Its readonly, no updates.

		perror []error
	}
)

var (
	opt bool = true
)

const (
	FILTER = "filter"
	ROOT   = "root"
)

//

func NewParser(l *lexer.Lexer) *Parser {
	p := &Parser{
		l: l,
	}
	// Read two tokens, to initialise curToken and peekToken
	p.nextToken()
	p.nextToken()
	//
	p.parseFns = make(map[token.TokenType]funcs.FuncT)
	p.registerFn(FILTER, token.EQ, funcs.FilterEQ)
	p.registerFn(ROOT, token.EQ, funcs.RootEQ)

	//
	// remove cacheClar before releasing..
	//
	//ast.CacheClear()
	return p
}

func (p *Parser) printToken(s ...string) {
	if len(s) > 0 {
		fmt.Printf("** Current Token: [%s] %s %s %v %s [%s]\n", s[0], p.curToken.Type, p.curToken.Literal, "Next Token:  ", p.peekToken.Type, p.peekToken.Literal)
	} else {
		fmt.Println("** Current Token: ", p.curToken.Type, p.curToken.Literal, "Next Token:  ", p.peekToken.Type, p.peekToken.Literal)
	}
}

func (p *Parser) hasError() bool {
	if len(p.perror) > 17 || p.abort {
		return true
	}
	return false
}

// addErr appends to error slice held in parser.
func (p *Parser) addErr(s string) error {
	if strings.Index(s, " at line: ") == -1 {
		s += fmt.Sprintf(" at line: %d, column: %d", p.curToken.Loc.Line, p.curToken.Loc.Col)
	}
	e := errors.New(s)
	p.perror = append(p.perror, e)
	return e
}

func (p *Parser) nextToken(s ...string) {
	p.curToken = p.peekToken

	p.peekToken = p.l.NextToken() // get another token from lexer:    [,+,(,99,Identifier,keyword etc.
	if len(s) > 0 {
		fmt.Printf("** Current Token: [%s] %s %s %s %s %s\n", s[0], p.curToken.Type, p.curToken.Literal, "Next Token:  ", p.peekToken.Type, p.peekToken.Literal)
	}
	if p.curToken != nil {
		if p.curToken.Illegal {
			p.addErr(fmt.Sprintf("Illegal %s token, [%s]", p.curToken.Type, p.curToken.Literal))
		}
	}
}

func (p *Parser) registerFn(funcType string, tokenType token.TokenType, fn parseFn) {
	p.parseFns[funcType+":"+tokenType] = fn
}

func (p *Parser) ParseFunction(s *FilterFunc) *Parser {

	// type FilterFunc struct {
	// 	parent *Expression
	// 	//value  bool   // for testing only
	// 	name string // for debug purposes - not used yet
	// 	//
	// 	// fname(predicate, value)
	// 	// fname(predFunc(predicate), value)
	// 	predicate string // represented by an nv at execution time
	// 	nv        *ds.NV
	// 	value     bool   // needs to be func() call
	// 	fname     string //      func(AttrName, AttrName, litVal, []DynaGValue, int) bool // eq,le,lt,gt,ge,allofterms, someofterms
	// 	predFunc  string // count(predicate), val(predicate)
	// }
	rf := s
	p.nextToken()
	//
	fmt.Printf("1: %#v\n", p.curToken)
	// eq, lt, gt, has, anyofterms, someofterms...
	if p.curToken.Type != token.RFUNC {
		p.addErr(fmt.Sprintf(`Expected a function  got %s instead`, p.curToken.Literal))
		return p
	}
	switch x := p.curToken.Literal; x {
	case token.EQ:
		rf.fname = func_[x]
		if f, ok := rootFunc[token.TokenType(x)]; !ok {
			p.addErr(fmt.Sprintf(`func %q is not registered`, p.curToken.Literal))
			return p
		} else {
			rf.F = f
		}
	case token.LT:
		rf.Name = x
		if f, ok := rootFunc[token.TokenType(x)]; !ok {
			p.addErr(fmt.Sprintf(`func %q is not registered`, p.curToken.Literal))
			return p
		} else {
			rf.F = f
		}
	}
	p.nextToken() // read over func
	// (
	if p.curToken.Type != token.LPAREN {
		p.addErr(fmt.Sprintf(`Expected (  got %s`, p.curToken.Literal))
		return p
	}
	p.nextToken() // read over (
	switch p.curToken.Type {

	case token.IDENT:
		rf.Predicate = p.curToken.Literal
		p.nextToken() // read over identier

	case token.MFUNC:
		rf.Modifier = p.curToken.Literal
		p.nextToken() // read over modifier
		if p.curToken.Type != token.LPAREN {
			p.addErr(fmt.Sprintf(`Expected (  got %s`, p.curToken.Literal))
			return p
		}
		p.nextToken() // read over (
		if p.curToken.Type != token.IDENT {
			p.addErr(fmt.Sprintf(`Expected identifier got %s`, p.curToken.Literal))
			return p
		}
		rf.Predicate = p.curToken.Literal
		p.nextToken() // read over identier
		p.nextToken() // read over )

	default:
		p.addErr(fmt.Sprintf(`Expected an identifier or modifer function got %s`, p.curToken.Literal))
		return p
	}
	// set default language
	//	s.Lang = "en"
	// if p.curToken.Type == token.ATSIGN {
	// 	p.nextToken() // read over @
	// 	if p.curToken.Type != token.IDENT {
	// 		p.addErr(fmt.Sprintf(`Expected a language identifer got %s instead`, p.curToken.Literal))
	// 		return
	// 	}
	// 	s.Lang = p.curToken.Literal
	// 	p.nextToken() // read over language
	// }
	//
	switch p.curToken.Type {
	case token.STRING:
		rf.Value = p.curToken.Literal
	case token.INT:
		i, _ := strconv.Atoi(p.curToken.Literal)
		rf.Value = i
	case token.FLOAT:
		f, err := strconv.ParseFloat(p.curToken.Literal, 64)
		if err != nil {
			p.addErr(fmt.Sprintf(`Errored in converting literal, %q, to float64. %s`, p.curToken.Literal, err.Error()))
			return p
		}
		rf.Value = f
	default:
		p.addErr(fmt.Sprintf(`Expected a string or number got %s instead`, p.curToken.Literal))
		return p
	}
	p.nextToken() // read over value

	for i := 0; i < 2; i++ {
		if p.curToken.Type != token.RPAREN {
			p.addErr(fmt.Sprintf(`Expected )  got %s instead`, p.curToken.Literal))
			return p
		}
		p.nextToken() // read over )
	}
	return p

}