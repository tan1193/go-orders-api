package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"go-order-service/internal/apperr"
	"go-order-service/internal/model"
)

type fakeRepo struct {
	orders       map[string]model.Order
	createCalled int
	listResp     []model.Order
	countResp    int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{orders: make(map[string]model.Order)}
}

func (f *fakeRepo) Create(_ context.Context, order model.Order) error {
	f.createCalled++
	f.orders[order.ID] = order
	return nil
}

func (f *fakeRepo) GetByID(_ context.Context, id string) (model.Order, error) {
	order, ok := f.orders[id]
	if !ok {
		return model.Order{}, apperr.ErrNotFound
	}
	return order, nil
}

func (f *fakeRepo) List(_ context.Context, _, _ int) ([]model.Order, error) {
	return f.listResp, nil
}

func (f *fakeRepo) Count(_ context.Context) (int, error) {
	return f.countResp, nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, id, status string) error {
	order, ok := f.orders[id]
	if !ok {
		return apperr.ErrNotFound
	}
	order.Status = status
	f.orders[id] = order
	return nil
}

type fakeEnqueuer struct {
	enqueued []string
}

func (f *fakeEnqueuer) Enqueue(_ context.Context, orderID string) error {
	f.enqueued = append(f.enqueued, orderID)
	return nil
}

func TestCreateOrder_Validation(t *testing.T) {
	tests := []struct {
		name         string
		customerName string
		amount       int
	}{
		{name: "empty customer name", customerName: "", amount: 100},
		{name: "blank customer name", customerName: "   ", amount: 100},
		{name: "zero amount", customerName: "Alice", amount: 0},
		{name: "negative amount", customerName: "Alice", amount: -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			enq := &fakeEnqueuer{}
			svc := NewOrderService(repo, enq)

			_, err := svc.CreateOrder(context.Background(), tc.customerName, tc.amount)
			if !errors.Is(err, apperr.ErrValidation) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestCreateOrder_PersistsAndEnqueues(t *testing.T) {
	repo := newFakeRepo()
	enq := &fakeEnqueuer{}
	svc := NewOrderService(repo, enq)

	order, err := svc.CreateOrder(context.Background(), "Alice", 120)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := uuid.Parse(order.ID); err != nil {
		t.Fatalf("expected valid UUID, got %s", order.ID)
	}
	if order.Status != model.StatusCreated {
		t.Fatalf("expected status %s, got %s", model.StatusCreated, order.Status)
	}
	if repo.createCalled != 1 {
		t.Fatalf("expected Create to be called once, got %d", repo.createCalled)
	}
	if len(enq.enqueued) != 1 || enq.enqueued[0] != order.ID {
		t.Fatalf("expected order ID to be enqueued once, got %#v", enq.enqueued)
	}
}

func TestListOrders_Validation(t *testing.T) {
	repo := newFakeRepo()
	enq := &fakeEnqueuer{}
	svc := NewOrderService(repo, enq)

	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{name: "limit zero", limit: 0, offset: 0},
		{name: "limit too high", limit: 101, offset: 0},
		{name: "negative offset", limit: 10, offset: -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := svc.ListOrders(context.Background(), tc.limit, tc.offset)
			if !errors.Is(err, apperr.ErrValidation) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestProcessOrder_TransitionsToCompleted(t *testing.T) {
	repo := newFakeRepo()
	enq := &fakeEnqueuer{}
	svc := NewOrderService(repo, enq)

	id := uuid.NewString()
	repo.orders[id] = model.Order{
		ID:           id,
		CustomerName: "Alice",
		Amount:       500,
		Status:       model.StatusCreated,
		CreatedAt:    time.Now(),
	}

	svc.delayFn = func() time.Duration { return 0 }
	svc.sleepFn = func(_ context.Context, _ time.Duration) error { return nil }

	if err := svc.ProcessOrder(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := repo.orders[id]
	if updated.Status != model.StatusCompleted {
		t.Fatalf("expected status %s, got %s", model.StatusCompleted, updated.Status)
	}
}
