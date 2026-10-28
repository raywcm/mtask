package service

import (
	"context"
	"github.com/raywcm/mtask/internal/monitor"
	"github.com/raywcm/mtask/internal/task"
)

func (m *Manager) Monitor(ctx context.Context, id task.ID, size task.Size) (*monitor.Session, error) {
	r, err := m.Store.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	return monitor.Open(ctx, m.Store, r, size), nil
}
