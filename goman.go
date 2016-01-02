// Copyright (c) 2016, Matt Davis following the permissive ISC license.
// See the LICENSE file that accompanies this software.
// https://www.isc.org/downloads/software-support-policy/isc-license/
//
// goman - Man page parsing library.
//
// Matt Davis: https://www.github.com/enferex/goman
package goman

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

// An option for the program that the man page describes.
// Often these are represented in the OPTIONS or SWITCHES section of a man page,
// and usually are prefixed with a '-' character.
type Opt struct {
	Name string
	Desc string
}

// ManPage represents the relevant fields of a man page.
// 'Opts' is a list of options provided by the man page.
type ManPage struct {
	Name     string
	Path     string
	Desc     string
	Synopsis string
	data     string
	Opts     []Opt
}

type parse_error struct {
	errmsg string
}

type macro struct {
	loc   []int
	mtype macro_type
}

type macro_type int

const (
	_ macro_type = iota
	b_macro
	ip_macro
	pp_macro
	sh_macro
	tp_macro
)

var macro_types = map[string]macro_type{
	"B":  b_macro,
	"IP": ip_macro,
	"PP": pp_macro,
	"SH": sh_macro,
	"TP": tp_macro,
}

func (pe *parse_error) Error() string {
	return pe.errmsg
}

// Given an offset return the next roff macro
func (man *ManPage) nextmacroOffset(offset int) *macro {
	re := regexp.MustCompilePOSIX(`^\.[A-Z]+ `)
	if idx := re.FindStringIndex(man.data[offset:]); idx != nil {
		index := []int{offset + idx[0], offset + idx[1]}
		str := man.data[index[0]+1 : index[1]-1]
		mt := macro_type(macro_types[str])
		return &macro{loc: index, mtype: mt}
	}
	return nil
}

// Return the next roff macro (nill if not found)
func (man *ManPage) nextmacro(macro *macro) *macro {
	return man.nextmacroOffset(macro.loc[1])
}

// Find the next roff section named 'name'
func (man *ManPage) findSection(name string) (int, *parse_error) {
	re := regexp.MustCompilePOSIX(`^\.SH *` + name)
	if idx := re.FindStringIndex(man.data); idx != nil {
		return idx[1], nil
	}
	return -1, &parse_error{"Error locating section"}
}

// Remove roff macros from a str
func stripmacros(str string) string {
	re := regexp.MustCompilePOSIX(`^\.[A-Z]+ *`)
	return re.ReplaceAllString(str, "")
}

// Return a string containing the roff section named 'sectname', or nil
// otherwise.
func (m *ManPage) getSection(sectname string) string {
	data := "N/A"
	if idx, err := m.findSection(sectname); err == nil {
		start := idx
		var mc *macro
		for mc = m.nextmacroOffset(start); mc != nil; mc = m.nextmacro(mc) {
			if mc.mtype == sh_macro {
				break
			}
		}
		if mc == nil {
			data = m.data[start:]
		} else {
			data = m.data[start:mc.loc[0]]
		}
		data = stripmacros(data)
	}
	return strings.TrimSpace(data)
}

func (m *ManPage) parseName() {
	name := strings.Split(m.getSection("NAME"), " ")[0]
	m.Name = strings.TrimRight(name, ` \,`)
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
	for mc := m.nextmacroOffset(idx); mc != nil; mc = m.nextmacro(mc) {
		if mc.mtype == sh_macro {
			break
		}

		if !(mc.mtype == b_macro || mc.mtype == ip_macro) {
			break
		}

		// B or IP
		opt := ""
		lines := strings.Split(m.data[mc.loc[1]:], "\n")
		for _, line := range lines {
			if len(line) == 0 || line[0] == '.' {
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

// Returns a string representation of an option specified in a man page.
func (o Opt) String() string {
	return o.Name + ": " + o.Desc
}

// Returns a string representation of a man page data structure.
func (m *ManPage) String() string {
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

// Instantiate and parse a man page given a gziped man page path.
func NewManPage(filename string) (*ManPage, error) {
	man := ManPage{Path: filename}

	fil, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Error opening man page: %v", err.Error())
	}
	defer fil.Close()

	rdr, err := gzip.NewReader(fil)
	if err != nil {
		return nil, fmt.Errorf("Error building a reader: ", err)
	}
	defer rdr.Close()

	data, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, fmt.Errorf("Error reading gzip data: ", err)
	}

	man.parse(string(data))
	return &man, nil
}
