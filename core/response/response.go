package response

import (
	"encoding/json"
	"fmt"
	"io"
)

type Envelope struct {
	OK    bool      `json:"ok"`
	Data  any       `json:"data,omitempty"`
	Error *ErrorObj `json:"error,omitempty"`
}

type ErrorObj struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

func WriteJSON(out io.Writer, data any) error {
	return write(out, Envelope{OK: true, Data: data})
}

func WriteError(out io.Writer, code, message, hint string) error {
	return write(out, Envelope{
		OK: false,
		Error: &ErrorObj{
			Code:    code,
			Message: message,
			Hint:    hint,
		},
	})
}

func write(out io.Writer, value Envelope) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s\n", encoded)
	return err
}
