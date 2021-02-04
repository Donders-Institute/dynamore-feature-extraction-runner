package util

import (
	"os"
	"testing"
)

func TestRunJob(t *testing.T) {
	cred, err := GetSyscallCredential("honlee")

	if err != nil {
		t.Error(err)
	}

	p := Payload{
		UserID:    "honlee",
		SessionID: "1",
		OutputDir: os.TempDir(),
	}

	if out, err := p.Run(cred); err != nil {
		t.Error(err)
	} else {
		t.Log(out)
	}
}
