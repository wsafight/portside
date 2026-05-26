package response

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteJSONWrapsData(t *testing.T) {
	var out bytes.Buffer
	if err := WriteJSON(&out, map[string]string{"status": "ok"}); err != nil {
		t.Fatal(err)
	}

	var envelope Envelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v\n%s", err, out.String())
	}
	if !envelope.OK {
		t.Fatalf("ok = false: %s", out.String())
	}
	if envelope.Data == nil {
		t.Fatalf("data missing: %s", out.String())
	}
}

func TestWriteErrorWrapsError(t *testing.T) {
	var out bytes.Buffer
	if err := WriteError(&out, "command_failed", "boom", "try again"); err != nil {
		t.Fatal(err)
	}

	var envelope Envelope
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v\n%s", err, out.String())
	}
	if envelope.OK {
		t.Fatalf("ok = true: %s", out.String())
	}
	if envelope.Error == nil {
		t.Fatalf("error missing: %s", out.String())
	}
	if envelope.Error.Code != "command_failed" || envelope.Error.Hint != "try again" {
		t.Fatalf("error = %#v", envelope.Error)
	}
}
