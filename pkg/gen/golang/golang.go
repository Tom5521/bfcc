// this package handles turning a brainfuck program into an output
// that can be compiled (generally C or ASM)
package golang

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"bfcc/pkg/lexer"
)

type Generator interface {
	// takes program input, and creates outfile
	Generate(input string, outpath string) error
}

type GolangGen struct {
	input   string
	output  string
	memsize uint
}

func New(memsize uint) *GolangGen {
	return &GolangGen{
		memsize: memsize,
	}
}

func (g *GolangGen) generateSrc() ([]byte, error) {
	var buf bytes.Buffer
	var start = `
/* 
* This program is auto-generated by bfcc
* sweetbbak :3
*/
package main

import (
	"os"
)

var array [%d]int
var idx int

func main() {

	inputFn := func() int {
		buf := make([]byte, 1)
		b, err := os.Stdin.Read(buf)
		if err != nil {
		    panic(err)
		}

		if b != 1 {
		    panic("byte not read")
		}

		return int(buf[0])
	}
	
	var b int
`

	// add memory size to header
	start = fmt.Sprintf(start, g.memsize)
	buf.WriteString(start)

	// create a lexer based on input
	l := lexer.New(g.input)

	// actual program parsing
	program := l.Tokens()

	token_index := 0
	for token_index < len(program) {
		// the current token
		tok := program[token_index]

		switch tok.Type {
		case lexer.INC_PTR:
			buf.WriteString(fmt.Sprintf("  idx += %d\n", tok.Repeat))
		case lexer.DEC_PTR:
			buf.WriteString(fmt.Sprintf("  idx -= %d\n", tok.Repeat))
		case lexer.INC_CELL:
			buf.WriteString(fmt.Sprintf("  array[idx] += %d\n", tok.Repeat))
		case lexer.DEC_CELL:
			buf.WriteString(fmt.Sprintf("  array[idx] -= %d\n", tok.Repeat))
		case lexer.OUTPUT:
			buf.WriteString("   os.Stdout.Write([]byte{byte(array[idx])})\n")

		case lexer.INPUT:
			str := `
			b = inputFn()
			array[idx] = b
`
			buf.WriteString(str)
		case lexer.LOOP_OPEN:
			// optimize [-] which loops and decrements a cell until it is zero by just setting it to zero 0x00 explicitly
			if token_index+2 < len(program) {
				// - and ]
				if program[token_index+1].Type == lexer.DEC_CELL && program[token_index+2].Type == lexer.LOOP_CLOSE {
					buf.WriteString("   array[idx] = 0\n")
					token_index += 3
					continue
				}
			}

			buf.WriteString("   for array[idx] != 0 {\n")
		case lexer.LOOP_CLOSE:
			buf.WriteString("}\n")
		default:
			// token not handled, decide what to do here
			// continue
			return nil, fmt.Errorf("unhandled token: %s at index %d", tok.Type, token_index)
		}
		token_index++
	}

	// close the main func
	buf.WriteString("}\n")

	return buf.Bytes(), nil
}

func (g *GolangGen) compileSrc(cpath string) error {

	go_build := exec.Command(
		"go",
		"build",
		"-o", g.output,
		"-ldflags",
		"-s -w",
		cpath,
	)

	go_build.Stdout = os.Stdout
	go_build.Stderr = os.Stderr

	return go_build.Run()
}

func (g *GolangGen) Generate(input string, output string) error {
	g.input = input
	g.output = output
	tmp := g.output + ".go"

	b, err := g.generateSrc()
	if err != nil {
		return err
	}

	err = os.WriteFile(tmp, b, 0o644)
	if err != nil {
		return err
	}

	err = g.compileSrc(tmp)
	if err != nil {
		return err
	}

	return nil
}
