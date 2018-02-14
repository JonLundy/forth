package main

import (
	"fmt"
	"io"
	"strings"
	"strconv"

	"sour.is/x/forth/naive"
	"sour.is/x/log"
	"github.com/chzyer/readline"
)


func main() {
	log.SetVerbose(log.Vinfo)

	f := InitForth()
	f.DStack = append(f.DStack, 0,0,0,0)
	fmt.Printf("%#v\n", f)
	f.Pages.Print()
	f.Read(BOOTSTRAP)
	return 

    l, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[31m»\033[0m ",
		HistoryFile:     "/tmp/readline.tmp",
		InterruptPrompt: "^C",
		EOFPrompt:       "bye",
		HistorySearchFold:   true,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()


	forth := naive.NewForth()
    for forth.State != naive.StateExit {
    	fmt.Printf("|  STATE: %s\n|  STACK: %s\n|  DICT: %v\n|  VARS: %v\n----\n\n", 
    		forth.State, forth.Stack, 
    		ikeys(forth.Dict), 
    		ikeys(forth.Vars))

		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			continue
			
		} else if err == io.EOF {
			break
		}

		lis := strings.Fields(line)
        err = forth.Execute(lis, 0)
        if err != nil {
        	log.Error(err)
        }
        fmt.Println("ok")
    }
}
func ikeys(m map[string]int64) (keys []string) {
	for key, _ := range m {
    	keys = append(keys, key)
	}
	return
}

func to_int(token string, base int) (int, bool) {
	if i, err := strconv.ParseInt(token, base, 64); err == nil {
		return int(i), true
	}
	return 0, false
}

type ForthWord struct {
	Name      string
	Page      *ForthPage
	Immediate bool
	Hidden    bool
	Native    bool
	Words     []int
}
type ForthPage struct {
	Parent    *ForthPage
	Offset    int
	Dict      []ForthWord
	Memory    []int
	Handler   ForthHandle
}
type ForthHandle func(ctx *AnnexiaForth, word ForthWord)
type WordPtr struct {
	Code  int
	Word  *ForthWord
	POS   int
	Page  *ForthPage
}
type AnnexiaForth struct {
	State     int
	Latest    int
	Here 	  int
	SZ 	      int
	Base      int

	RSP       WordPtr
	RStack    []WordPtr
	DStack    []int
	Pages     *ForthPage

	POS       int
	Input     []string
}

func InitForth() (f *AnnexiaForth) {
	f = &AnnexiaForth{Base: 10}
	p := AddPage(nil, RootHandler)

	p.DefCode("DOCOL")

	// Easy FORTH Primitives
	p.DefCode("DROP")
	p.DefCode("SWAP")
	p.DefCode("DUP")
	p.DefCode("OVER")
	p.DefCode("ROT")
	p.DefCode("-ROT")
	p.DefCode("2DROP")
	p.DefCode("2DUP")
	p.DefCode("2SWAP")
	p.DefCode("?DUP")
	p.DefCode("1+")
	p.DefCode("1-")
	p.DefCode("4+")
	p.DefCode("4-")
	p.DefCode("+")
	p.DefCode("-")
	p.DefCode("*")
	p.DefCode("/MOD")

	// Comparison Ops
	p.DefCode("=")
	p.DefCode("<>")
	p.DefCode("<")
	p.DefCode(">")
	p.DefCode("<=")
	p.DefCode(">=")
	p.DefCode("0=")
	p.DefCode("0<>")
	p.DefCode("0<")
	p.DefCode("0>")
	p.DefCode("0<=")
	p.DefCode("0>=")
	p.DefCode("AND")
	p.DefCode("OR")
	p.DefCode("XOR")
	p.DefCode("INVERT")
	
	
	// Literals
	p.DefCode("LIT")
	p.DefCode("LITSTRING")
	p.DefCode("TELL")

	// Memory
	p.DefCode("!")
	p.DefCode("@")
	p.DefCode("+!")
	p.DefCode("-!")

	// Built-in Variables
	p.DefCode("STATE")
	p.DefCode("HERE")
	p.DefCode("LATEST")
	p.DefCode("S0")
	p.DefCode("BASE")

	// Built-in Constants
	p.DefCode("VERSION")
	p.DefCode("R0")
	p.DefCode("_DOCOL")
	p.DefCode("__F_IMMED")
	p.DefCode("__F_HIDDEN")
	p.DefCode("__F_LENMASK")

	// Return Stack
	p.DefCode(">R")
	p.DefCode("R>")
	p.DefCode("RSP@")
	p.DefCode("RSP!")
	p.DefCode("RDROP")
	
	// Data Stack
	p.DefCode("DSP@")
	p.DefCode("DSP!")
	
	// Input and Output
	p.DefCode("KEY")
	p.DefCode("EMIT")
	p.DefCode("WORD")
	p.DefCode("NUMBER")
	
	// Dictionary Ops
	p.DefCode("FIND")
	p.DefCode(">CFA")
	
	// Compiling
	p.DefCode("CREATE")
	p.DefCode(",")
	p.DefCode(".")
	
	// Immediate
	p.DefCode("[").SetImmediate()
	p.DefCode("]")
	p.DefCode("IMMEDIATE").SetImmediate()
	p.DefCode("HIDDEN")
	p.DefCode("'")
	
	// Branching
	p.DefCode("BRANCH")
	p.DefCode("0BRANCH")
	
	// Interpreting
	p.DefCode("INTERPRET")
	p.DefCode("EXIT")
	p.DefCode("CHAR")
	p.DefCode("EXECUTE")

	p = AddPage(p, RootHandler)
	f.Pages = p

	// Misc. Words
	p.DefWord("DOUBLE",    "DUP +", f.Base)
	p.DefWord("QUADRUPLE", "DOUBLE DOUBLE", f.Base)
	p.DefWord(">DFA",      ">CFA 1+", f.Base)
	p.DefWord("COLON",     "WORD CREATE LIT DOCOL , LATEST @ HIDDEN [", f.Base)
	p.DefWord("SEMICOLON", "LIT , LATEST @ HIDDEN ]", f.Base).SetImmediate()
	p.DefWord("HIDE",      "WORD FIND HIDDEN", f.Base)
	p.DefWord("QUIT",      "R0 RSP! INTERPRET BRANCH -1", f.Base)
	f.Latest = p.Offset + len(p.Dict)

	return
}
func AddPage(p *ForthPage, h ForthHandle) (np *ForthPage){
	
	np = &ForthPage{Parent: p, Handler: h}
	if p != nil {
		np.Offset = p.Offset + len(p.Dict)
		np.Memory = append([]int(nil), p.Memory...)
	}

	return
}
func (p *ForthPage) DefCode(name string) (*ForthWord){
	code := ForthWord{Name:name, Native: true, Page: p}
	p.Dict = append(p.Dict, code)
	return &code
}
func (p *ForthPage) DefWord(name, words string, base int) (*ForthWord){
	var w []int
	pos, _ := p.FindWord("DOCOL", base)
	w = append(w, pos)
	for _, word := range strings.Fields(words) {
		pos, _ = p.FindWord(word, base)
		w = append(w, pos)
	}
	pos, _ = p.FindWord("EXIT", base)
	w = append(w, pos)

	code := ForthWord{Name:name, Native: false, Words: w, Page: p}
	p.Dict = append(p.Dict, code)
	return &code
}
func (w *ForthWord) SetImmediate() (*ForthWord){
	w.Immediate = !w.Immediate
	return w
}
func (w *ForthWord) SetHidden() (*ForthWord){
	w.Hidden = !w.Hidden
	return w
}
func (p *ForthPage) FindWord(name string, base int) (int, *ForthWord) {
	name = strings.ToUpper(name)

	for {
		if v, ok := to_int(name, base); ok {
			return v, nil
		}
		for w := len(p.Dict)-1; w >= 0; w-- {
			if p.Dict[w].Hidden {
				continue
			}
			if name == p.Dict[w].Name {
				return p.Offset + w, &p.Dict[w]
			}
		}
		if p.Parent == nil {
			break
		}
		p = p.Parent
	}

	return 0, nil
}
func (p *ForthPage) FindCode(code int) (int, *ForthWord) {
	if code < 0 {
		return 0, nil
	}
	if code > p.Offset + len(p.Dict) {
		return 0, nil
	}

	for {
		if code < p.Offset {
			if p.Parent == nil {
				break
			}
			p = p.Parent
		} else {
			return code, &p.Dict[code - p.Offset]
		}
	}

	return 0, nil
}
func RootHandler(ctx *AnnexiaForth, w ForthWord) {
	switch w.Name {
	case "DOCOL":
		fmt.Println(ctx.RSP, ctx.RStack)
		//ctx.RSP.POS += 1
		//ctx.RStack = append(ctx.RStack, ctx.RSP)
		//log.Info(ctx.RStack)

	case "EXIT":

	case "DROP":
		ctx.DStack = ctx.DStack[:len(ctx.DStack)-1]	
	case "SWAP":
		ctx.DStack[len(ctx.DStack)-2], ctx.DStack[len(ctx.DStack)-1] = 
			ctx.DStack[len(ctx.DStack)-1], ctx.DStack[len(ctx.DStack)-2]
	case "DUP":
		ctx.DStack = append(ctx.DStack, ctx.DStack[len(ctx.DStack)-1])
	case "OVER":
		ctx.DStack = append(ctx.DStack, ctx.DStack[len(ctx.DStack)-2])
	case "ROT":
		a, b, c := ctx.DStack[len(ctx.DStack)-1], ctx.DStack[len(ctx.DStack)-2], ctx.DStack[len(ctx.DStack)-3]
		ctx.DStack[len(ctx.DStack)-1], ctx.DStack[len(ctx.DStack)-2], ctx.DStack[len(ctx.DStack)-3] = b, c, a
	case "-ROT":
		a, b, c := ctx.DStack[len(ctx.DStack)-1], ctx.DStack[len(ctx.DStack)-2], ctx.DStack[len(ctx.DStack)-3]
		ctx.DStack[len(ctx.DStack)-1], ctx.DStack[len(ctx.DStack)-2], ctx.DStack[len(ctx.DStack)-3] = a, c, b
	case "2DROP":
		ctx.DStack = ctx.DStack[:len(ctx.DStack)-2]
	case "2DUP":
		ctx.DStack = append(ctx.DStack, ctx.DStack[len(ctx.DStack)-2], ctx.DStack[len(ctx.DStack)-1])
	case "2SWAP":
		a, b, c, d := ctx.DStack[len(ctx.DStack)-1], ctx.DStack[len(ctx.DStack)-2], ctx.DStack[len(ctx.DStack)-3], ctx.DStack[len(ctx.DStack)-4]
		ctx.DStack[len(ctx.DStack)-1], ctx.DStack[len(ctx.DStack)-2], ctx.DStack[len(ctx.DStack)-3], ctx.DStack[len(ctx.DStack)-4] = c, d, a, b 
	case "?DUP":
		if ctx.DStack[len(ctx.DStack)-1] != 0 {
			ctx.DStack = append(ctx.DStack, ctx.DStack[len(ctx.DStack)-1])
		}

	case "1+":
		i := ctx.DStack[len(ctx.DStack)-1]
		ctx.DStack[len(ctx.DStack)-1] = i + 1
	case "1-":
		i := ctx.DStack[len(ctx.DStack)-1]
		ctx.DStack[len(ctx.DStack)-1] = i - 1

	case "4+":
		i := ctx.DStack[len(ctx.DStack)-1]
		ctx.DStack[len(ctx.DStack)-1] = i + 4
	case "4-":
		i := ctx.DStack[len(ctx.DStack)-1]
		ctx.DStack[len(ctx.DStack)-1] = i - 4

	case "+":
		n := ctx.DStack[len(ctx.DStack)-2]
		i := ctx.DStack[len(ctx.DStack)-1]
		ctx.DStack = ctx.DStack[:len(ctx.DStack)-1]
		ctx.DStack[len(ctx.DStack)-1] = n + i
	case "-":
		n := ctx.DStack[len(ctx.DStack)-2]
		i := ctx.DStack[len(ctx.DStack)-1]
		ctx.DStack = ctx.DStack[:len(ctx.DStack)-1]
		ctx.DStack[len(ctx.DStack)-1] = n - i
	case "*":
		n := ctx.DStack[len(ctx.DStack)-2]
		i := ctx.DStack[len(ctx.DStack)-1]
		ctx.DStack = ctx.DStack[:len(ctx.DStack)-1]
		ctx.DStack[len(ctx.DStack)-1] = n * i
	case "/MOD":
		n := ctx.DStack[len(ctx.DStack)-2]
		d := ctx.DStack[len(ctx.DStack)-1]
		ctx.DStack = ctx.DStack[:len(ctx.DStack)-1]
		ctx.DStack[len(ctx.DStack)-2] = n % d
		ctx.DStack[len(ctx.DStack)-1] = n / d

	case "SPACES":
		n := ctx.DStack[len(ctx.DStack)-1]
		fmt.Print(strings.Repeat(" ", n))
	case "EMIT":
		c := ctx.DStack[len(ctx.DStack)-1]
		fmt.Printf("%c", c)
		ctx.DStack = ctx.DStack[:len(ctx.DStack)-1]
	case ".":
		fmt.Printf("POP: %d\n", ctx.DStack[len(ctx.DStack)-1])
		ctx.DStack = ctx.DStack[:len(ctx.DStack)-1]
	default:
		log.Error("Word Not Implemented: %s", w.Name)
	}
}
func (f *AnnexiaForth) Read(in string) {
	f.Input = strings.Fields(in)

	for f.POS = 0; f.POS < len(f.Input);  {
		code, word := f.Pages.FindWord(f.Input[f.POS], f.Base)

	//	fmt.Printf("%s %d %+v", f.Input[f.POS], code, word)
		
		f.RSP = WordPtr{Code: code, Word: word, Page: f.Pages}

		for f.RSP.Word != nil {
			c := *f.RSP.Word
			f.RSP = f.RSP.Next();
			
			fmt.Print("CURRENT ", c, " NEXT ", f.RSP.Word)
			fmt.Println(f.DStack)
			fmt.Println(f.RStack)

			if c.Native {
				c.Page.Handler(f, c)
			} 

		}

		f.POS++
	}
}
func (wp WordPtr) Next() (WordPtr) {
	fmt.Printf("POS: %d\n", wp.POS)

	if wp.Word == nil {
		return WordPtr{}
	}
	if wp.POS >= len(wp.Word.Words) {
		return WordPtr{}
	}
	code, word := wp.Page.FindCode(wp.Word.Words[wp.POS])

	return WordPtr{Code: code, Word: word, POS: wp.POS, Page: wp.Page}
}

func (p *ForthPage) Print() {
	for {
		for w := len(p.Dict)-1; w >= 0; w-- {
			if p.Dict[w].Hidden {
				continue
			}
			fmt.Println("WORD ", w + p.Offset, p.Dict[w].Name)
		}
		if p.Parent == nil {
			break
		}
		p = p.Parent
	}
}




var BOOTSTRAP string = `
1+ 1+ DOUBLE .
1+ 1+ QUADRUPLE .
4+ DOUBLE DOUBLE 1+ .
`
/*  
  : /    /MOD SWAP DROP ;
  : MOD  /MOD DROP ;
  : '\n' 10   ;
  : BL   32   ;
  : CR   '\N' EMIT ;
  : SPACE BL EMIT ;
  : NEGATE 0 SWAP - ;
  : TRUE 1 ;
  : FALSE 0 ;
  : NIP  SWAP DROP ;
  : TUCK SWAP OVER ;
  : DOUBLE    DUP + ;
  : QUADRUPLE DOUBLE DOUBLE ;
  : DECIMAL 10 BASE ! ;
  : HEX     16 BASE ! ;
  : ? ( addr -- ) @ . ;
  : ':' [ CHAR : ] LITERAL ;
  : ';' [ CHAR ; ] LITERAL ;
  : '(' [ CHAR ( ] LITERAL ;
  : ')' [ CHAR ) ] LITERAL ;
  : '"' [ CHAR " ] LITERAL ;
  : 'A' [ CHAR A ] LITERAL ;
  : '0' [ CHAR 0 ] LITERAL ;
  : '-' [ CHAR - ] LITERAL ;
  : '.' [ CHAR . ] LITERAL ;
`
*/