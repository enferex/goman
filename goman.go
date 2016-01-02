package main

import (
	"compress/gzip"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Opt struct {
	name string
	desc string
}

type ManPage struct {
	name     string
	path     string
	desc     string
	synopsis string
	data     string
	opts     []Opt
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

func NewDB() *sql.DB {
	db, err := sql.Open("sqlite3", "manpages.db")
	if err != nil {
		log.Fatal("Error opening database ", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS man
                     (id INTEGER PRIMARY KEY,
                     name TEXT UNIQUE,
                     path TEXT UNIQUE,
                     description TEXT,
                     synopsis TEXT)`)
	if err != nil {
		log.Fatal("Error creating man page table ", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS options
                     (id INTEGER PRIMARY KEY,
                      man_id      INT,
                      name        TEXT,
                      description TEXT,
                      type        TEXT)`)
	if err != nil {
		log.Fatal("Error creating options table ", err)
	}

	return db
}

func safeString(str string) string {
	str = strconv.QuoteToASCII(str)
    str = strings.Replace(str, "'", "''", -1)
	return "'" + str[1:len(str)-1] + "'"
}

func (man *ManPage) addToDB(db *sql.DB) {
	n := safeString(man.name)
	p := safeString(man.path)
	d := safeString(man.desc)
	s := safeString(man.synopsis)
	q := "REPLACE INTO man (name, path, description, synopsis) "
	q += fmt.Sprintf(`VALUES(%s,%s,%s,%s)`, n, p, d, s)

    var res sql.Result
    var err error
	if res, err = db.Exec(q); err != nil {
		log.Fatal("Error adding db entry ", err)
	}

    var man_id int64
	if man_id, err = res.LastInsertId(); err != nil {
		log.Fatal("Error obtaining last inserted row ID ", err)
    }

	for _, opt := range man.opts {
		n := safeString(opt.name)
		d := safeString(opt.desc)
		q := "REPLACE INTO options"
		q += "(man_id, name, description, type) "
		q += fmt.Sprintf("VALUES(%d, %s, %s, %s)", man_id, n, d, `'N/A'`)
		if _, err := db.Exec(q); err != nil {
			log.Fatal("Error adding man page option into db ", err)
		}
	}
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
    m.name = strings.Split(m.getSection("NAME"), " ")[0]
}

func (m *ManPage) parseDesc() {
	m.desc = m.getSection("DESCRIPTION")
}

func (m *ManPage) parseSynopsis() {
	m.synopsis = m.getSection("SYNOPSIS")
}

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
				m.opts = append(m.opts, Opt{name: opt_name, desc: opt_desc})
			}
		}
	}
}

func (o Opt) String() string {
	return o.name + ": " + o.desc
}

func (m ManPage) String() string {
	str := fmt.Sprintf(
		"Name: %s\n"+
			"Desc:     %s\n"+
			"Synposis: %s\n", m.name, m.desc, m.synopsis)
	str += "Options:\n"
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
	man.parseName()
	man.parseDesc()
	man.parseSynopsis()
	man.parseOpts()
}

func NewManPage(name string) *ManPage {
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
	return &man
}

func main() {
	var filename string
	var add_to_db bool
	flag.StringVar(&filename, "f", "", "man page to parse")
	flag.BoolVar(&add_to_db, "d", false, "Add manpage information to database")
	flag.Parse()
	println("Parsing " + filename + "...")

	db := NewDB()
	defer db.Close()
	man := NewManPage(filename)
	if add_to_db {
		man.addToDB(db)
	}
}
