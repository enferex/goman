package goman

import (
	"testing"
)

func TestNewManPage(t *testing.T) {
	man, err := NewManPage("./test.1.gz") // Use the dummy testing man page
	if err != nil {
		t.Errorf(err.Error())
	}
	name := "foobar"
	if man.Name != name {
		t.Errorf("Name: expected '%s', found '%s'\n", name, man.Name)
	}

	desc := `This is just a sample based on the example provided by http://www.tldp.org/HOWTO/Man-Page/q3.html`
	if man.Desc != desc {
		t.Errorf("Desc: expected '%s', found '%s'\n", desc, man.Desc)
	}

	opts := []Opt{
		{"-q", "q is an option"},
		{"-u", "u is an option"},
		{"-x", "x is an option"},
	}
	for i, opt := range opts {
		if man.Opts[i] != opt {
			t.Errorf("Opts: expected '%s', found '%s'\n", opt, man.Opts[i])
		}
	}

}
