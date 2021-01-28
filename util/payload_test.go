package util

import (
	"testing"
)

func TestRunJob(t *testing.T) {
	cred, err := GetSyscallCredential("honlee")

	if err != nil {
		t.Error(err)
	}

	p := Payload{}

	if out, err := p.Run(cred); err != nil {
		t.Error(err)
	} else {
		t.Log(out)
	}
}
