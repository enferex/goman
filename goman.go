package goman

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

type Opt struct {
	Name string
	Desc string
}

type ManPage struct {
	Name     string
	Path     string
	Desc     string
	Synopsis string
	data     string
	Opts     []Opt
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

// Given an offset return the next roff macro
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

// Return the next roff macro (nill if not found)
func (man *ManPage) nextMacro(macro *Macro) *Macro {
	return man.nextMacroOffset(macro.loc[1])
}

// Find the next roff section named 'name'
func (man *ManPage) findSection(name string) (int, *ParseError) {
	re := regexp.MustCompilePOSIX(`^\.SH *` + name)
	if idx := re.FindStringIndex(man.data); idx != nil {
		return idx[1], nil
	}
	return -1, &ParseError{"Error locating section"}
}

// Remove roff macros from a str
func stripMacros(str string) {
	re := regexp.MustCompilePOSIX(`^\.[A-Z]+ *`)
	str = re.ReplaceAllString(str, "")
}

// Return a string containing the roff section named 'sectname', or nil
// otherwise.
func (m *ManPage) getSection(sectname string) string {
	data := "N/A"
	if idx, err := m.findSection(sectname); err == nil {
		start := idx
		var mc *Macro
		for mc = m.nextMacroOffset(start); mc != nil; mc = m.nextMacro(mc) {
			if mc.mtype == SH {
				break
			}
		}
		if mc == nil {
			data = m.data[start:]
		} else {
			data = m.data[start:mc.loc[0]]
		}
		stripMacros(data)
	}
	return strings.TrimSpace(data)
}

func (m *ManPage) parseName() {
	m.Name = strings.Split(m.getSection("NAME"), " ")[0]
}

func (m *ManPage) parseDesc() {
	m.Desc = m.getSection("DESCRIPTION")
}

func (m *ManPage) parseSynopsis() {
	m.Synopsis = m.getSection("SYNOPSIS")
}

// Parse out options from the man page
func (m *ManPage) parseOpts() {
	idx, err := m.findSection(`(OPTIONS|SWITCHES)`)
	if err != nil {
		if idx, err = m.findSection(`DESCRIPTION`); err != nil {
			return
		}
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
				m.Opts = append(m.Opts, Opt{Name: opt_name, Desc: opt_desc})
			}
		}
	}
}

func (o Opt) String() string {
	return o.Name + ": " + o.Desc
}

func (m ManPage) String() string {
	str := fmt.Sprintf(
		"Name: %s\n"+
			"Desc:     %s\n"+
			"Synposis: %s\n", m.Name, m.Desc, m.Synopsis)
	str += "Options:\n"
	for _, o := range m.Opts {
		str += fmt.Sprintf("%v\n", o)
	}

	return str
}

func (man *ManPage) parse(data string) {
	// Remove carriage return
	replace := strings.NewReplacer("\x0D", "")
	man.data = replace.Replace(data)

	// Parse all of the interesting parts
	man.parseName()
	man.parseDesc()
	man.parseSynopsis()
	man.parseOpts()
}

// Instantiate and parse a manpage given a gziped manpage path.
func NewManPage(filename string) *ManPage {
	man := ManPage{Path: filename}

	fil, err := os.Open(filename)
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
	return &man
}
