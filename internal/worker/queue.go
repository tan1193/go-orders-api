package worker

import (
	"context"
	"fmt"
)

type Enqueuer interface {
	Enqueue(ctx context.Context, orderID string) error
}

type ChannelEnqueuer struct {
	Ch chan<- string
}

func (e ChannelEnqueuer) Enqueue(ctx context.Context, orderID string) error {
	select {
	case e.Ch <- orderID:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("enqueue order id=%s: %w", orderID, ctx.Err())
	}
}
