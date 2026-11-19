package guest

import (
	"bytes"
	"testing"
)

func TestEmbeddedPowerShell(t *testing.T) {
	scripts, err := Scripts()
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) != 3 || len(scripts[EntryScript]) == 0 {
		t.Fatal("incomplete embedded scripts")
	}
	for name, data := range scripts {
		for _, b := range data {
			if b > 127 {
				t.Errorf("%s must remain ASCII for Windows PowerShell 5.1", name)
				break
			}
		}
		if bytes.Contains(bytes.ReplaceAll(data, []byte("\r\n"), nil), []byte("\n")) {
			t.Errorf("%s requires CRLF", name)
		}
	}
}
