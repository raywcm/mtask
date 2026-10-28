// Package service is the shared application boundary for CLI and TUI.
package service

import (
	"context"
	"fmt"
	"github.com/raywcm/mtask/internal/i18n"
	"github.com/raywcm/mtask/internal/task"
	"strconv"
	"time"
)

type Repository interface {
	List(context.Context) ([]task.Record, error)
	Load(context.Context, task.ID) (task.Record, error)
	Create(context.Context, task.Spec) error
	SaveRuntime(context.Context, task.Runtime) error
}
type TaskList interface {
	List(context.Context) ([]task.Record, error)
}
type Catalog struct{ Store Repository }

func (c Catalog) List(ctx context.Context) ([]task.Record, error) {
	if c.Store == nil {
		return nil, fmt.Errorf("task repository is not configured")
	}
	return c.Store.List(ctx)
}
func Label(r task.Record) string { return LabelLocale(r, i18n.New("en")) }
func LabelLocale(r task.Record, l i18n.Locale) string {
	state := "not_started"
	if r.Runtime != nil {
		state = string(r.Runtime.State)
	}
	if r.Health == "unreachable" {
		state = "unreachable"
	}
	id := string(r.Spec.ID)
	if len(id) > 8 {
		id = id[:8]
	}
	d := r.Elapsed(time.Now())
	seconds := int64(d.Seconds())
	return fmt.Sprintf("%s  %s  %s  %02d:%02d:%02d", id, strconv.QuoteToGraphic(r.Spec.Name), l.Text(state), seconds/3600, seconds/60%60, seconds%60)
}
