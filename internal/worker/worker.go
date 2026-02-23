package worker

import (
	"context"
	"log"
)
// interface in consumer side
type Processor interface {
	ProcessOrder(ctx context.Context, orderID string) error
}

type OrderWorker struct {
	Queue     <-chan string
	Processor Processor
	Logger    *log.Logger
}

func NewOrderWorker(queue <-chan string, processor Processor, logger *log.Logger) *OrderWorker {
	return &OrderWorker{Queue: queue, Processor: processor, Logger: logger}
}

func (w *OrderWorker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.Logger.Printf("msg=worker_stopped reason=context_canceled")
			return
		case orderID, ok := <-w.Queue:
			if !ok {
				w.Logger.Printf("msg=worker_stopped reason=queue_closed")
				return
			}
			if err := w.Processor.ProcessOrder(ctx, orderID); err != nil {
				w.Logger.Printf("msg=worker_process_failed order_id=%s err=%q", orderID, err)
			}
		}
	}
}
