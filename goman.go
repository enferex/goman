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
	_ MacroType = iota
	B
	IP
	PP
	SH
	TP
)

var MacroTypes = map[string]MacroType{
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

func (m *ManPage) parseOpts() {
	idx, err := m.findSection(`(OPTIONS|SWITCHES)`)
	if err != nil {
		return
	}

	// We have a OPTIONS or SWITCHES section
	for mc := m.nextMacroOffset(idx); mc != nil; mc = m.nextMacro(mc) {
		if mc.mtype == SH {
			break
		}

		if !(mc.mtype == B || mc.mtype == IP) {
			break
		}

		// B or IP
		opt := ""
		lines := strings.Split(m.data[mc.loc[1]:], "\n")
		for _, line := range lines {
			if line[0] == '.' {
				break
			}
			opt += " " + line
		}

		// Grab '-<optname>\n'
		opt = strings.TrimLeft(opt, " ")
		if idx := strings.Index(opt, "-"); idx != -1 {
			if spc := strings.IndexAny(opt[idx:], "\r "); spc != -1 {
                spc += 1
				opt_name := opt[idx:spc]
				opt_desc := strings.Trim(opt[spc:], " ")
				m.opts = append(m.opts, Opt{name: opt_name, desc: opt_desc})
			}
		}
	}
}

func (o Opt) String() string {
	return o.name + ": " + o.desc
}

func (m ManPage) String() string {
	str := fmt.Sprintf("Name: %s\n"+
		"Desc: %s\n", m.name, m.desc)
	for _, o := range m.opts {
		str += fmt.Sprintf("%v\n", o)
	}

	return str
}

func (man *ManPage) parse(data string) {
	// Remove carriage return
	replace := strings.NewReplacer("\x0D", "")
	man.data = replace.Replace(data)

	// Parse all of the interesting parts
	man.parseDesc()
	man.parseOpts()
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
