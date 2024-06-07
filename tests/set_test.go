package tests

import (
	"bytes"
	"fmt"
	"testing"
)



func TestSet(t *testing.T) {
	var b []byte
	var buf = bytes.NewBuffer(b)
	conn := getLocalConnection()
	for i := 1; i < 100; i++ {
		buf.WriteByte('a')
		cmd := fmt.Sprintf("SET k%d %s", i, buf.String())
		fireCommand(conn, cmd)
	}

	for i := 1; i < 100; i++ {
		cmd := fmt.Sprintf("GET k%d", i)
		v := fireCommand(conn, cmd)
		if len(v.(string)) != i {
			t.Fail()
		}
	}
	fireCommand(conn, "ABORT")
}
