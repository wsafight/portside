package tui

import (
	"fmt"
	"io"
)

// Run starts Portside's terminal interface. It is intentionally part of the
// main portside CLI instead of a second standalone product.
func Run(out io.Writer) error {
	_, err := fmt.Fprintln(out, "Portside TUI is reserved as the next CLI experience.")
	return err
}
