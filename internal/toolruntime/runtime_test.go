package toolruntime

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type testCloser struct {
	err error
}

func (c testCloser) Close() error {
	return c.err
}

func TestCleanupStackRunsInReverseOrderAndJoinsAllFailures(t *testing.T) {
	firstErr := errors.New("first cleanup")
	secondErr := errors.New("second cleanup")
	var calls []string

	stack := CleanupStack{}
	stack.AddDirectory("first", func(path string) error {
		calls = append(calls, path)
		return firstErr
	})
	stack.AddFile("second", func(path string) error {
		calls = append(calls, path)
		return secondErr
	})

	err := stack.Run(false)
	if !reflect.DeepEqual(calls, []string{"second", "first"}) {
		t.Fatalf("cleanup calls = %#v, want reverse registration order", calls)
	}
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("cleanup error = %v, want both failures", err)
	}
}

func TestCleanupStackHonorsKeep(t *testing.T) {
	called := false
	stack := CleanupStack{}
	stack.AddFile("artifact", func(string) error {
		called = true
		return nil
	})

	if err := stack.Run(true); err != nil {
		t.Fatalf("Run(true) error = %v", err)
	}
	if called {
		t.Fatal("Run(true) removed artifact")
	}
}

func TestJoinPrimaryAndCleanupErrorsPreservesErrorsIs(t *testing.T) {
	primaryErr := errors.New("primary")
	cleanupErr := errors.New("cleanup")

	err := JoinPrimaryAndCleanupErrors(primaryErr, cleanupErr)
	if !errors.Is(err, primaryErr) || !errors.Is(err, cleanupErr) {
		t.Fatalf("joined error = %v, want primary and cleanup", err)
	}
}

func TestCloseAllJoinsAllFailures(t *testing.T) {
	firstErr := errors.New("first close")
	secondErr := errors.New("second close")

	err := CloseAll([]testCloser{{err: firstErr}, {}, {err: secondErr}})
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("CloseAll() error = %v, want both failures", err)
	}
}

func TestOperationContextUsesTimeout(t *testing.T) {
	ctx, cancel := OperationContext(context.Background(), time.Nanosecond)
	defer cancel()

	<-ctx.Done()
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("ctx.Err() = %v, want deadline exceeded", ctx.Err())
	}
}
