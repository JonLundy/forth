package naive

import (
	"fmt"
	"strings"
	"strconv"

	"sour.is/x/log"
)

/*
	package main

	import (
		"fmt"
		"io"
		"strings"

		"sour.is/x/forth/naive"
		"sour.is/x/log"
		"github.com/chzyer/readline"
	)


	func main() {
		log.SetVerbose(log.Vinfo)

		forth := naive.NewForth()
		forth.Execute(strings.Fields(naive.BOOTSTRAP), 0)

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
*/

var BOOTSTRAP string = `
." Staring Bootstrap"
  : /    ( n d -- d ) /MOD SWAP DROP ;
  : MOD  ( n d -- r ) /MOD DROP ;
  : '\n' 10   ;
  : BL   32   ;
  : CR   '\N' EMIT ;
  : SPACE BL EMIT ;
  : NEGATE 0 SWAP - ;
  : TRUE 1 ;
  : FALSE 0 ;
  : NIP ( x y -- y ) SWAP DROP ;
  : TUCK ( x y -- y x y ) SWAP OVER ;
  : DOUBLE ( x -- 2x ) DUP + ;
  : QUADRUPLE ( x -- 4x ) DOUBLE DOUBLE ;
  : DECIMAL ( -- ) 10 BASE ! ;
  : HEX ( -- ) 16 BASE ! ;
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

  : TEST
	DEPTH . CR

	42 DUP . . CR
	23 DROP DEPTH . CR
	1 2 SWAP . . CR
	1 2 OVER . . . CR
	1 2 3 -ROT . . . CR
	1 2 3 ROT . . . CR
	1 2 3 4 2DROP . . CR
	1 2 3 4 2DUP . . . . . . CR
	1 2 3 4 2SWAP . . . . CR

	DEPTH . CR
  ;

." Completed Bootstrap"
`

type ForthState int

const (
	StateInterpret ForthState = iota
	StateDefinition
	StateCompile
	StateComment
	StateDotQuote
	StateSee
	StateExit
)  

type Forth struct {
	State   ForthState
	Stack   []string
	QStack  []string
	DStack  []string
	Dict    map[string]int64
	Vars    map[string]int64
	Memory  []string
}

func NewForth() (f *Forth) {
	log.Info("Initializing new Forth")

	f = new(Forth)
	f.State = StateInterpret
	
	f.Dict = make(map[string]int64)
	f.Vars = make(map[string]int64)
	f.Vars["BASE"] = 10
	f.Memory = append(f.Memory, "BYE")

	return
}

func (fs ForthState) String() string {
	switch fs {
	case StateInterpret:  return "StateInterpret"
	case StateDefinition: return "StateDefinition"
	case StateCompile:    return "StateCompile"
	case StateComment:    return "StateComment"
	case StateDotQuote:   return "StateDotQuote"
	case StateSee:        return "StateSee"
	default:              return "UNKNOWN"
	}
}

func (f *Forth) Execute(lis []string, start int64) (err error) {
	name := ""
	var RStack []int64

START:
	for rsp := start; rsp < int64(len(lis));  {
		token := lis[rsp]
		TOKEN := strings.ToUpper(token)

		switch(f.State) {
		case StateDefinition:
			name = strings.ToUpper(token)
			log.Debugf("Begin Definition for %s", name)
			f.State = StateCompile

		case StateCompile:
			switch(TOKEN) {
			case "LITERAL":
				var c string
				c, f.DStack = f.DStack[len(f.DStack)-1], f.DStack[:len(f.DStack)-1] 
				f.DStack = append(f.DStack, "LIT", c)

			case "CHAR":
				rsp++
				f.DStack = append(f.DStack, lis[rsp])

			case ":":
				return fmt.Errorf(": INSIDE :")

			case ";":
				log.Debugf("Complete Definition for %s", name)
				i := int64(len(f.Memory))
				f.Memory = append(f.Memory, f.DStack...)
				f.Memory = append(f.Memory, "NEXT")
				f.Dict[name] = i
				f.DStack = nil
				f.State = StateInterpret
			case "[":
			case "]":

			default:
				f.DStack = append(f.DStack, token)
			}

		case StateComment:
			if token == ")" {
				f.State = StateInterpret
			}

		case StateDotQuote:
			if strings.HasSuffix(token, `"`) {
				f.QStack = append(f.QStack, token[:len(token)-1])
				log.Infof("QUOTE: %s", strings.Join(f.QStack, " "))
				f.QStack = nil
				f.State = StateInterpret
			} else {
				f.QStack = append(f.QStack, token)
			}

		case StateSee:
			f.State = StateInterpret
			if v, ok := f.Dict[TOKEN]; ok {
				var see []string
				for _, t := range f.Memory[v:] {
					if t == "NEXT" {
						break
					}
					see = append(see, t)
				}
				fmt.Println(see)
			} else {
				return fmt.Errorf("UNKN TOKEN: %s", token)
			}

		case StateInterpret:
			switch(TOKEN){
			case ":":
				f.State = StateDefinition
			case "(":
				f.State = StateComment
			case `."`:
				f.State = StateDotQuote
			case "LIT":
				rsp++
				f.Stack = append(f.Stack, lis[rsp])
			case "SEE":
				f.State = StateSee
			case "DEPTH":
				f.Stack = append(f.Stack, fmt.Sprintf("i%d", len(f.Stack)))
			case "!":
				if len(f.Stack) < 2 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				dst := f.Stack[len(f.Stack)-1]
				v, ok := to_int(f.Stack[len(f.Stack)-2], 10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
				f.Vars[dst] = v
				f.Stack = f.Stack[:len(f.Stack)-2]
				
			case "@":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				src := f.Stack[len(f.Stack)-1]
				v, ok := f.Vars[src]
				if !ok {
					return fmt.Errorf("Variable not in memory: %s", f.Stack[len(f.Stack)-1])
				}
				log.Info(v, ok)

				f.Stack[len(f.Stack)-1] = fmt.Sprintf("i%d", v)
			case ".":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				var v string
				f.Stack, v = f.Stack[:len(f.Stack)-1], f.Stack[len(f.Stack)-1]
				if i, ok := to_int(v, 10); ok {
					if f.Vars["BASE"] == 10 {
						fmt.Printf("%d\n", i)
					} else {
						fmt.Printf("0x%x\n", i)
					}
				} else {
					fmt.Println("POP:", v)
				}
			case "NEXT":
				if len(RStack) == 0 {
					return nil
				} else {
					RStack, rsp = RStack[:len(RStack)-1], RStack[len(RStack)-1]
				}
 			case "BYE":
				f.State = StateExit
				return nil
			case "DROP":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				f.Stack = f.Stack[:len(f.Stack)-1]
			case "SWAP":
				if len(f.Stack) < 2 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				f.Stack[len(f.Stack)-2], f.Stack[len(f.Stack)-1] = f.Stack[len(f.Stack)-1], f.Stack[len(f.Stack)-2]
			case "DUP":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				f.Stack = append(f.Stack, f.Stack[len(f.Stack)-1])
			case "OVER":
				if len(f.Stack) < 2 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				f.Stack = append(f.Stack, f.Stack[len(f.Stack)-2])
			case "ROT":
				if len(f.Stack) < 3 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				a, b, c := f.Stack[len(f.Stack)-1], f.Stack[len(f.Stack)-2], f.Stack[len(f.Stack)-3]
				f.Stack[len(f.Stack)-1], f.Stack[len(f.Stack)-2], f.Stack[len(f.Stack)-3] = b, c, a
			case "-ROT":
				if len(f.Stack) < 3 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				a, b, c := f.Stack[len(f.Stack)-1], f.Stack[len(f.Stack)-2], f.Stack[len(f.Stack)-3]
				f.Stack[len(f.Stack)-1], f.Stack[len(f.Stack)-2], f.Stack[len(f.Stack)-3] = a, c, b
			case "2DROP":
				if len(f.Stack) < 2 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				f.Stack = f.Stack[:len(f.Stack)-2]
			case "2DUP":
				if len(f.Stack) < 2 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				f.Stack = append(f.Stack, f.Stack[len(f.Stack)-2], f.Stack[len(f.Stack)-1])
			case "2SWAP":
				if len(f.Stack) < 4 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				a, b, c, d := f.Stack[len(f.Stack)-1], f.Stack[len(f.Stack)-2], f.Stack[len(f.Stack)-3], f.Stack[len(f.Stack)-4]
				f.Stack[len(f.Stack)-1], f.Stack[len(f.Stack)-2], f.Stack[len(f.Stack)-3], f.Stack[len(f.Stack)-4] = c, d, a, b 
			case "?DUP":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				if f.Stack[len(f.Stack)-1] != "i0" {
					f.Stack = append(f.Stack, f.Stack[len(f.Stack)-1])
				}
			case "1+":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				i, ok := to_int(f.Stack[len(f.Stack)-1],10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
				f.Stack[len(f.Stack)-1] = fmt.Sprintf("i%d", i + 1)
			case "1-":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				i, ok := to_int(f.Stack[len(f.Stack)-1],10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
				f.Stack[len(f.Stack)-1] = fmt.Sprintf("i%d", i - 1)
			case "4+":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				i, ok := to_int(f.Stack[len(f.Stack)-1],10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
				f.Stack[len(f.Stack)-1] = fmt.Sprintf("i%d", i + 4)
			case "4-":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				i, ok := to_int(f.Stack[len(f.Stack)-1],10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
				f.Stack[len(f.Stack)-1] = fmt.Sprintf("i%d", i - 4)
			case "+":
				if len(f.Stack) < 2 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				n, ok := to_int(f.Stack[len(f.Stack)-1], 10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
				i, ok := to_int(f.Stack[len(f.Stack)-2], 10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-2])
				}
				f.Stack = f.Stack[:len(f.Stack)-1]
				f.Stack[len(f.Stack)-1] = fmt.Sprintf("i%d", n + i)
			case "-":
				if len(f.Stack) < 2 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				n, ok := to_int(f.Stack[len(f.Stack)-1], 10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
				i, ok := to_int(f.Stack[len(f.Stack)-2], 10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-2])
				}
				f.Stack = f.Stack[:len(f.Stack)-1]
				f.Stack[len(f.Stack)-1] = fmt.Sprintf("i%d", i - n)
			case "*":
				if len(f.Stack) < 2 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				var ok bool
				var n, i int64
				if n, ok = to_int(f.Stack[len(f.Stack)-1], 10); !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
				if i, ok = to_int(f.Stack[len(f.Stack)-2], 10); !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-2])
				}
				f.Stack = f.Stack[:len(f.Stack)-1]
				f.Stack[len(f.Stack)-1] = fmt.Sprintf("i%d", i * n)
			case "/MOD":
				if len(f.Stack) < 2 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				d, ok := to_int(f.Stack[len(f.Stack)-1], 10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
				n, ok := to_int(f.Stack[len(f.Stack)-2], 10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-2])
				}
				if d == 0 {
					return fmt.Errorf("Div by Zero")
				}
				f.Stack[len(f.Stack)-2] = fmt.Sprintf("i%d", n % d)
				f.Stack[len(f.Stack)-1] = fmt.Sprintf("i%d", n / d)
			case "SPACES":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				n, ok := to_int(f.Stack[len(f.Stack)-1], 10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
				fmt.Print(strings.Repeat(" ", int(n)))
			case "EMIT":
				if len(f.Stack) < 1 {
					return fmt.Errorf("Insufficent Stack Size")
				}
				c, ok := to_int(f.Stack[len(f.Stack)-1], 10)
				if !ok {
					return fmt.Errorf("Non integer value on stack: %s", f.Stack[len(f.Stack)-1])
				}
					fmt.Printf("%c", c)
				f.Stack = f.Stack[:len(f.Stack)-1]

			default:
				// log.Debug("Fallthrough to dict/vars")
				if v, ok := to_int(token, f.Vars["BASE"]); ok {
					// log.Debugf("PUSH: %d", v)
					f.Stack = append(f.Stack, fmt.Sprintf("i%d", v))
				
				} else if _, ok := f.Vars[TOKEN]; ok {
					f.Stack = append(f.Stack, TOKEN)

				} else if v, ok := f.Dict[TOKEN]; ok {
					log.Debugf("Executing: %s @ %d", TOKEN, v)

					if start == 0 {
						err = f.Execute(f.Memory, int64(v))
					} else {
						RStack = append(RStack, rsp)
						rsp = int64(v)
						continue START
					}

					if err != nil {
						return err
					}
				} else {
					return fmt.Errorf("UNKN TOKEN: %s", TOKEN)
				}
			}

		default:
			return fmt.Errorf("UNKN STATE: %s", f.State)
		}

		rsp++
	}

	return
}

func to_int(token string, base int64) (int64, bool) {
	if token[0] == 'i' {
		token = token[1:]
		base = 10
	}

	if i, err := strconv.ParseInt(token, int(base), 64); err == nil {
		return i, true
	}
	return 0, false
}

func to_hex(n int) string {
	return fmt.Sprintf("%X", n)
}