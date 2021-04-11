package main

import "context"

type Collector interface {
	Collect(ctx context.Context) error
}
