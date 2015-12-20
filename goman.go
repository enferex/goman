package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

type Opt struct {
	name string
	desc string
}

type ManPage struct {
	name string
	desc string
	data string
	opts []Opt
}

type ParseError struct {
	errmsg string
}

type Macro struct {
	loc   []int
	mtype MacroType
}

type MacroType int

const (
	B MacroType = iota
	IP
	PP
	SH
	TP
)

var MacroTypes = map[string]MacroType {
    "B":  B,
    "IP": IP,
    "PP": PP,
    "SH": SH,
    "TP": TP,
}

func (pe *ParseError) Error() string {
	return pe.errmsg
}

func (man *ManPage) nextMacroOffset(offset int) *Macro {
	re := regexp.MustCompilePOSIX(`^\.[A-Z]+ `)
	if idx := re.FindStringIndex(man.data[offset:]); idx != nil {
		index := []int{offset + idx[0], offset + idx[1]}
        str := man.data[index[0]+1 : index[1]-1]
		mt := MacroType(MacroTypes[str])
		return &Macro{loc: index, mtype: mt}
	}
	return nil
}

func (man *ManPage) nextMacro(macro *Macro) *Macro {
	return man.nextMacroOffset(macro.loc[1])
}

func (man *ManPage) findSection(name string) (int, *ParseError) {
	re := regexp.MustCompilePOSIX(`^\.SH *` + name)
	if idx := re.FindStringIndex(man.data); idx != nil {
		return idx[1], nil
	}
	return -1, &ParseError{"Error locating section"}
}

func stripMacros(str string) {
    re := regexp.MustCompilePOSIX(`^\.[A-Z]+ *`)
    str = re.ReplaceAllString(str, "")
}

func (m *ManPage) parseDesc() {
	m.desc = "N/A"
	if idx, err := m.findSection("DESCRIPTION"); err == nil {
		start := idx
        var mc *Macro
		for mc = m.nextMacroOffset(start); mc != nil; mc = m.nextMacro(mc) {
			if mc.mtype == SH {
				break
			}
		}
        if mc == nil {
            m.desc = m.data[start:]
        } else {
            m.desc = m.data[start:mc.loc[0]]
        }
        stripMacros(m.desc)
	}
}

func (man ManPage) String() string {
	return fmt.Sprintf("Name: %s\n" +
		"Desc: %s\n", man.name, man.desc)
}

func (man *ManPage) parse(data string) {
	// Remove carriage return
	replace := strings.NewReplacer("\x0D", "")
	man.data = replace.Replace(data)

	// Parse all of the interesting parts
	man.parseDesc()
	//man.parseOpts()
}

func NewManPage(name string) ManPage {
	man := ManPage{name: name}

	fil, err := os.Open(name)
	if err != nil {
		log.Fatal("Error opening man page", err)
	}
	defer fil.Close()

	rdr, err := gzip.NewReader(fil)
	if err != nil {
		log.Fatal("Error building a reader ", err)
	}
	defer rdr.Close()

	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		log.Fatal("Error reading gzip data ", err)
	}

	man.parse(string(data))

	return man
}

func main() {
	var filename string
	flag.StringVar(&filename, "f", "", "man page to parse")
	flag.Parse()
	println("Parsing " + filename + "...")

	man := NewManPage(filename)
	fmt.Print(man)
}
