package service

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"

	"go-order-service/internal/apperr"
	"go-order-service/internal/model"
	"go-order-service/internal/repository"
	"go-order-service/internal/worker"
)

type OrderService struct {
	repo    repository.OrderRepository
	queue   worker.Enqueuer
	delayFn func() time.Duration
	sleepFn func(ctx context.Context, d time.Duration) error
}

func NewOrderService(repo repository.OrderRepository, queue worker.Enqueuer) *OrderService {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &OrderService{
		repo:  repo,
		queue: queue,
		delayFn: func() time.Duration {
			return time.Duration(300+r.Intn(501)) * time.Millisecond
		},
		sleepFn: sleepWithContext,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, customerName string, amount int) (model.Order, error) {
	if strings.TrimSpace(customerName) == "" {
		return model.Order{}, fmt.Errorf("customer_name is required: %w", apperr.ErrValidation)
	}
	if amount <= 0 {
		return model.Order{}, fmt.Errorf("amount must be greater than 0: %w", apperr.ErrValidation)
	}

	order := model.Order{
		ID:           uuid.NewString(),
		CustomerName: strings.TrimSpace(customerName),
		Amount:       amount,
		Status:       model.StatusCreated,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return model.Order{}, fmt.Errorf("create order: %w", err)
	}

	if err := s.queue.Enqueue(ctx, order.ID); err != nil {
		return model.Order{}, fmt.Errorf("enqueue order id=%s: %w", order.ID, err)
	}

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (model.Order, error) {
	if strings.TrimSpace(id) == "" {
		return model.Order{}, fmt.Errorf("id is required: %w", apperr.ErrValidation)
	}
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return model.Order{}, fmt.Errorf("get order id=%s: %w", id, err)
	}
	return order, nil
}

func (s *OrderService) ListOrders(ctx context.Context, limit, offset int) ([]model.Order, int, error) {
	if limit <= 0 || limit > 100 {
		return nil, 0, fmt.Errorf("limit must be 1..100: %w", apperr.ErrValidation)
	}
	if offset < 0 {
		return nil, 0, fmt.Errorf("offset must be >= 0: %w", apperr.ErrValidation)
	}

	orders, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	return orders, total, nil
}

func (s *OrderService) ProcessOrder(ctx context.Context, orderID string) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("load order before processing id=%s: %w", orderID, err)
	}

	if order.Status != model.StatusCreated {
		return nil
	}

	if err := s.repo.UpdateStatus(ctx, orderID, model.StatusProcessing); err != nil {
		return fmt.Errorf("set processing id=%s: %w", orderID, err)
	}

	if err := s.sleepFn(ctx, s.delayFn()); err != nil {
		return fmt.Errorf("simulate processing id=%s: %w", orderID, err)
	}

	if err := s.repo.UpdateStatus(ctx, orderID, model.StatusCompleted); err != nil {
		return fmt.Errorf("set completed id=%s: %w", orderID, err)
	}

	return nil
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
