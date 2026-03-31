package vector

import "context"

// Noop 永不命中，用于默认集成或未接库。
type Noop struct{}

func (Noop) Search(ctx context.Context, in SearchInput) (SearchResult, bool, error) {
	return SearchResult{}, false, nil
}
