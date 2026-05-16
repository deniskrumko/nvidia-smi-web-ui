package output

import (
	"encoding/json"
	"io"
)

// WriteJSON writes a stable, indented JSON representation.
func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
