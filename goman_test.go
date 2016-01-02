package goman

import "testing"

func TestNewManPage(t *testing.T) {
	_, err := NewManPage("./test.1.gz") // Use the dummy testing man page
	if err != nil {
		t.Errorf(err.Error())
	}
}
