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
    loc []int
    mtype MacroType
}

type MacroType string

const (
  TP MacroType = "TP"
  SH MacroType = "SH"
  PP MacroType = "PP"
)

func (pe *ParseError) Error() string {
    return pe.errmsg
}

func (man *ManPage) nextMacro(offset int) (*Macro, *ParseError) {
    re := regexp.MustCompilePOSIX(`^\.`)
    if idx := re.FindStringIndex(man.data[offset:]); idx != nil {
        index := []int{offset + idx[0], offset + idx[1]}
        mt := MacroType("TP")
        return &Macro{loc:index, mtype:mt} , nil
    }
    return nil, &ParseError{"Error locating next macro"}
}

func (man *ManPage) findSection(name string) (int, *ParseError) {
	re := regexp.MustCompilePOSIX(`^\.SH.*` + name)
	if idx := re.FindStringIndex(man.data); idx != nil {
        return idx[1], nil
    }
    return -1, &ParseError{"Error locating section"}
}

func (man *ManPage) parseDesc() {
	man.desc = "N/A"
	if idx, err := man.findSection("DESCRIPTION"); err == nil {
		start := idx
        if macro, err := man.nextMacro(start); err == nil {
			man.desc = man.data[start : macro.loc[1]]
        }
	}
}

func (man ManPage) String() string {
	return fmt.Sprintf("Name: %s\n"+
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
