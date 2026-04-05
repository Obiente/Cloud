package databases

import "testing"

func TestClassifySQLQuery(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		queryType string
		readOnly  bool
		wantErr   bool
	}{
		{
			name:      "select with comments",
			query:     "/* comment */ SELECT * FROM users -- trailing",
			queryType: "SELECT",
			readOnly:  true,
		},
		{
			name:      "update query",
			query:     "UPDATE users SET active = true",
			queryType: "UPDATE",
			readOnly:  false,
		},
		{
			name:    "multiple statements rejected",
			query:   "SELECT 1; SELECT 2",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryType, readOnly, err := classifySQLQuery(tt.query)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if queryType != tt.queryType {
				t.Fatalf("expected query type %q, got %q", tt.queryType, queryType)
			}
			if readOnly != tt.readOnly {
				t.Fatalf("expected readOnly=%v, got %v", tt.readOnly, readOnly)
			}
		})
	}
}

func TestBuildQueryResultRow(t *testing.T) {
	row := buildQueryResultRow(
		[]string{"id", "name", "payload", "empty"},
		[]interface{}{int64(42), "hello", []byte("world"), nil},
	)

	if len(row.Cells) != 4 {
		t.Fatalf("expected 4 cells, got %d", len(row.Cells))
	}
	if row.Cells[0].GetValue() != "42" {
		t.Fatalf("expected numeric value to be formatted, got %q", row.Cells[0].GetValue())
	}
	if row.Cells[1].GetValue() != "hello" {
		t.Fatalf("expected string value, got %q", row.Cells[1].GetValue())
	}
	if row.Cells[2].GetValue() != "world" {
		t.Fatalf("expected []byte value to be formatted, got %q", row.Cells[2].GetValue())
	}
	if !row.Cells[3].GetIsNull() {
		t.Fatalf("expected nil value to be marked null")
	}
}
