package util

import (
	"os"
	"testing"
)

func TestRunJob(t *testing.T) {
	p := Payload{
		UserID:    "honlee",
		SessionID: "1",
		OutputDir: os.TempDir(),
	}

	if out, err := p.Run("honlee"); err != nil {
		t.Error(err)
	} else {
		t.Log(out)
	}
}
