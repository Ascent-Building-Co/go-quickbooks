package quickbooks

import (
	"fmt"
	"strings"
)

// QueryMaxPageSize is the maximum number of results the QBO API will return
// in a single query response. Pass this to WithPageSize or WithPage to
// request the largest possible page.
const QueryMaxPageSize = 1000

type QueryOption interface {
	applyQuery(*queryConfig)
}

type queryOptionFunc func(*queryConfig)

func (f queryOptionFunc) applyQuery(cfg *queryConfig) {
	f(cfg)
}

type queryConfig struct {
	startPosition int
	pageSize      int
}

func WithStartPosition(startPosition int) QueryOption {
	return queryOptionFunc(func(cfg *queryConfig) {
		cfg.startPosition = startPosition
	})
}

func WithPageSize(pageSize int) QueryOption {
	return queryOptionFunc(func(cfg *queryConfig) {
		cfg.pageSize = pageSize
	})
}

// WithPage sets both STARTPOSITION and MAXRESULTS from a 1-based page cursor.
// Page 1 starts at position 1, page 2 at pageSize+1, and so on. It is a
// convenience alternative to combining WithStartPosition and WithPageSize
// when iterating through results page by page.
func WithPage(page, pageSize int) QueryOption {
	startPosition := ((page - 1) * pageSize) + 1
	return queryOptionFunc(func(cfg *queryConfig) {
		cfg.startPosition = startPosition
		cfg.pageSize = pageSize
	})
}

func applyQueryOptions(query string, opts ...QueryOption) string {
	var cfg queryConfig
	for _, opt := range opts {
		opt.applyQuery(&cfg)
	}

	query = strings.TrimSpace(query)
	upper := strings.ToUpper(query)

	if cfg.startPosition > 0 && !strings.Contains(upper, "STARTPOSITION") {
		query = fmt.Sprintf("%s STARTPOSITION %d", query, cfg.startPosition)
	}
	if cfg.pageSize > 0 && !strings.Contains(upper, "MAXRESULTS") {
		query = fmt.Sprintf("%s MAXRESULTS %d", query, cfg.pageSize)
	}

	return strings.TrimSpace(query)
}
