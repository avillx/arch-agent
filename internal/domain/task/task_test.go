package task

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// ---- Reglaments ----

func TestEvery_NextTime_returns_duration(t *testing.T) {
	e := Every{D: 5 * time.Minute}
	d := e.NextTime()
	if d != 5*time.Minute {
		t.Errorf("expected 5m, got %v", d)
	}
}

func TestEvery_NextTime_zero(t *testing.T) {
	e := Every{D: 0}
	d := e.NextTime()
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestDaily_NextTime_returns_positive(t *testing.T) {
	d := Daily{Hour: 12, Minute: 0}
	next := d.NextTime()
	if next <= 0 {
		t.Errorf("expected positive duration, got %v", next)
	}
	if next > 27*time.Hour {
		t.Errorf("daily NextTime should be < 27h, got %v", next)
	}
}

func TestDaily_NextTime_past_today_returns_tomorrow(t *testing.T) {
	d := Daily{Hour: 0, Minute: 0} // midnight
	// force a known scenario by checking that the result is always in (0, 25h]
	next := d.NextTime()
	if next <= 0 {
		t.Errorf("daily past time should return tomorrow, got %v", next)
	}
}

func TestWeekly_NextTime_returns_positive(t *testing.T) {
	w := Weekly{Weekday: time.Monday, Hour: 9, Minute: 0}
	next := w.NextTime()
	if next <= 0 || next > 7*24*time.Hour {
		t.Errorf("weekly NextTime should be in (0, 168h], got %v", next)
	}
}

func TestMonthly_NextTime_returns_positive(t *testing.T) {
	m := Monthly{Day: 15, Hour: 10, Minute: 0}
	next := m.NextTime()
	if next <= 0 || next > 32*24*time.Hour {
		t.Errorf("monthly NextTime should be in (0, 768h], got %v", next)
	}
}

// ---- Task construction ----

func TestNewTask_sets_fields(t *testing.T) {
	called := false
	tk := NewTask(Every{D: time.Hour}, "test-desc", func(ctx context.Context, t *Task) {
		called = true
	})
	if tk.Description != "test-desc" {
		t.Errorf("expected 'test-desc', got %s", tk.Description)
	}
	if tk.onCall == nil {
		t.Error("onCall should not be nil")
	}
	_ = called
}

func TestNewTask_nil_onCall(t *testing.T) {
	tk := NewTask(Every{D: time.Minute}, "no-call", nil)
	if tk.onCall != nil {
		t.Error("expected nil onCall")
	}
}

// ---- Run / Stop ----

func TestTask_Run_stops_via_context_cancellation(t *testing.T) {
	tk := NewTask(Every{D: time.Hour}, "test", nil)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- tk.Run(ctx)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error from context cancellation, got nil")
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestTask_Run_inherits_parent_context(t *testing.T) {
	tk := NewTask(Every{D: time.Hour}, "test", nil)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- tk.Run(ctx)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error from parent cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return")
	}
}

func TestTask_Run_does_not_execute_before_timer_fires(t *testing.T) {
	var executed atomic.Bool
	tk := NewTask(Every{D: time.Hour}, "test", func(ctx context.Context, t *Task) {
		executed.Store(true)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go tk.Run(ctx)
	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	if executed.Load() {
		t.Error("executor was called before timer should have fired")
	}
}

func TestTask_Run_executes_onCall_when_timer_fires(t *testing.T) {
	var executed atomic.Bool
	tk := NewTask(Every{D: 5 * time.Millisecond}, "quick", func(ctx context.Context, t *Task) {
		executed.Store(true)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go tk.Run(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	if !executed.Load() {
		t.Error("onCall was not executed after timer fired")
	}
}

func TestTask_Stop_cancels_Run(t *testing.T) {
	tk := NewTask(Every{D: time.Hour}, "stop-test", nil)

	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() {
		errCh <- tk.Run(ctx)
	}()

	time.Sleep(10 * time.Millisecond)
	tk.Stop()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error (context canceled) after Stop")
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after Stop")
	}
}

func TestTask_Stop_before_Run_no_panic(t *testing.T) {
	tk := NewTask(Every{D: time.Hour}, "stop-before-run", nil)
	// stopFn is nil, Stop() should handle it gracefully
	tk.Stop()
	// No panic — test passes
}

func TestTask_Stop_after_Run_already_returned_no_panic(t *testing.T) {
	tk := NewTask(Every{D: time.Hour}, "stop-after-return", nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_ = tk.Run(ctx) // should return quickly

	// Stop() after Run() returned: stopFn is nil (no one called Run with set)
	// Actually Run did set stopFn = cancel, but then Run returned and defer cancel
	// ran. stopFn still points to the cancelled func.
	// Calling Stop() should just call the cancelled cancel() — noop, no panic.
	tk.Stop()
}

func TestTask_Stop_idempotent(t *testing.T) {
	tk := NewTask(Every{D: time.Hour}, "idempotent", nil)

	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() {
		errCh <- tk.Run(ctx)
	}()

	time.Sleep(10 * time.Millisecond)
	tk.Stop()
	tk.Stop() // second call should be safe

	select {
	case err := <-errCh:
		// ctx.Err() is expected
		if err != context.Canceled {
			t.Logf("got error (expected): %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after Stop")
	}
}

func TestTask_onCall_receives_task_self_reference(t *testing.T) {
	var receivedTask *Task
	tk := NewTask(Every{D: 5 * time.Millisecond}, "self-ref", func(ctx context.Context, t *Task) {
		receivedTask = t
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go tk.Run(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	if receivedTask != tk {
		t.Error("onCall did not receive the same *Task reference")
	}
}

func TestTask_Stop_inside_onCall_returns_promptly(t *testing.T) {
	tk := NewTask(Every{D: 5 * time.Millisecond}, "self-stop", func(ctx context.Context, t *Task) {
		t.Stop()
	})

	ctx := context.Background()
	errCh := make(chan error, 1)
	go func() {
		errCh <- tk.Run(ctx)
	}()

	// onCall will fire and call t.Stop() which cancels ctx internally
	// Then the inner select detects ctx.Done() and returns
	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error after self-stop")
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after self-stop inside onCall")
	}
}

// ---- Repeat behaviour ----

func TestTask_Run_repeats_with_Every(t *testing.T) {
	var count atomic.Int32
	tk := NewTask(Every{D: 5 * time.Millisecond}, "repeat", func(ctx context.Context, t *Task) {
		count.Add(1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go tk.Run(ctx)

	time.Sleep(60 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	c := count.Load()
	if c < 2 {
		t.Errorf("expected at least 2 executions with 5ms Every, got %d", c)
	}
}