package quickbooks

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyQueryOptions(t *testing.T) {
	tests := []struct {
		name  string
		query string
		opts  []QueryOption
		want  string
	}{
		{
			name:  "no options",
			query: "SELECT * FROM Invoice",
			want:  "SELECT * FROM Invoice",
		},
		{
			name:  "start position only",
			query: "SELECT * FROM Invoice",
			opts:  []QueryOption{WithStartPosition(11)},
			want:  "SELECT * FROM Invoice STARTPOSITION 11",
		},
		{
			name:  "page size only",
			query: "SELECT * FROM Invoice",
			opts:  []QueryOption{WithPageSize(10)},
			want:  "SELECT * FROM Invoice MAXRESULTS 10",
		},
		{
			name:  "start position and page size",
			query: "SELECT * FROM Invoice",
			opts:  []QueryOption{WithStartPosition(11), WithPageSize(10)},
			want:  "SELECT * FROM Invoice STARTPOSITION 11 MAXRESULTS 10",
		},
		{
			name:  "does not duplicate STARTPOSITION already in query",
			query: "SELECT * FROM Invoice STARTPOSITION 5",
			opts:  []QueryOption{WithStartPosition(11)},
			want:  "SELECT * FROM Invoice STARTPOSITION 5",
		},
		{
			name:  "does not duplicate MAXRESULTS already in query",
			query: "SELECT * FROM Invoice MAXRESULTS 50",
			opts:  []QueryOption{WithPageSize(10)},
			want:  "SELECT * FROM Invoice MAXRESULTS 50",
		},
		{
			name:  "trims surrounding whitespace",
			query: "  SELECT * FROM Invoice  ",
			opts:  []QueryOption{WithPageSize(5)},
			want:  "SELECT * FROM Invoice MAXRESULTS 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyQueryOptions(tt.query, tt.opts...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWithPage(t *testing.T) {
	tests := []struct {
		name              string
		page              int
		pageSize          int
		wantStartPosition int
		wantMaxResults    int
	}{
		{
			name:              "first page",
			page:              1,
			pageSize:          10,
			wantStartPosition: 1,
			wantMaxResults:    10,
		},
		{
			name:              "second page",
			page:              2,
			pageSize:          10,
			wantStartPosition: 11,
			wantMaxResults:    10,
		},
		{
			name:              "third page",
			page:              3,
			pageSize:          25,
			wantStartPosition: 51,
			wantMaxResults:    25,
		},
		{
			name:              "large page number",
			page:              100,
			pageSize:          20,
			wantStartPosition: 1981,
			wantMaxResults:    20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyQueryOptions("SELECT * FROM Invoice", WithPage(tt.page, tt.pageSize))
			assert.Contains(t, got, fmt.Sprintf("STARTPOSITION %d", tt.wantStartPosition))
			assert.Contains(t, got, fmt.Sprintf("MAXRESULTS %d", tt.wantMaxResults))
		})
	}
}

// TestWithPageMatchesWithStartPositionAndPageSize confirms that WithPage
// produces the same query as the equivalent WithStartPosition + WithPageSize
// combination.
func TestWithPageMatchesWithStartPositionAndPageSize(t *testing.T) {
	tests := []struct {
		page     int
		pageSize int
	}{
		{page: 1, pageSize: 10},
		{page: 2, pageSize: 10},
		{page: 5, pageSize: 25},
	}

	for _, tt := range tests {
		startPosition := ((tt.page - 1) * tt.pageSize) + 1
		withPage := applyQueryOptions("SELECT * FROM Invoice", WithPage(tt.page, tt.pageSize))
		explicit := applyQueryOptions("SELECT * FROM Invoice", WithStartPosition(startPosition), WithPageSize(tt.pageSize))
		assert.Equal(t, explicit, withPage, "page=%d pageSize=%d", tt.page, tt.pageSize)
	}
}
