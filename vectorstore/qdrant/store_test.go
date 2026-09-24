package qdrant

import (
	"encoding/json"
	"testing"
)

func TestPointIDUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{name: "UUID string", json: `"de2be4aa-e719-4ab7-b89b-7eb6036043ab"`, want: "de2be4aa-e719-4ab7-b89b-7eb6036043ab"},
		{name: "numeric ID", json: `42`, want: "42"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var id pointID
			if err := json.Unmarshal([]byte(test.json), &id); err != nil {
				t.Fatalf("unmarshal point ID: %v", err)
			}
			if got := id.String(); got != test.want {
				t.Fatalf("point ID = %q, want %q", got, test.want)
			}
		})
	}
}
