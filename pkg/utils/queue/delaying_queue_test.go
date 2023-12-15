/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package queue

import (
	"context"
	"testing"
	"time"

	fakeclock "k8s.io/utils/clock/testing"

	"sigs.k8s.io/kwok/pkg/utils/wait"
)

func TestSimpleAddAfter(t *testing.T) {
	fakeClock := fakeclock.NewFakeClock(time.Now())
	dq := NewDelayingQueue[string](fakeClock)

	first := "foo"
	dq.AddAfter(first, 50*time.Millisecond)

	// Simulate clock time passed
	fakeClock.Step(10 * time.Millisecond)
	if dq.Len() != 0 {
		t.Fatalf("should not have added")
	}

	fakeClock.Step(100 * time.Millisecond)
	err := checkAdded(dq, 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCancel(t *testing.T) {
	fakeClock := fakeclock.NewFakeClock(time.Now())
	dq := NewDelayingQueue[string](fakeClock)

	first := "foo"

	dq.AddAfter(first, 500*time.Millisecond)

	fakeClock.Step(100 * time.Millisecond)
	dq.Cancel(first)

	fakeClock.Step(1 * time.Second)

	if dq.Len() != 0 {
		t.Fatalf("expected empty")
	}
}

// TestAddTwo tests the case where the former added element has a larger delayed duration than
// the element added later.
func TestAddTwo(t *testing.T) {
	fakeClock := fakeclock.NewFakeClock(time.Now())
	dq := NewDelayingQueue[string](fakeClock)

	first := "foo"
	second := "bar"

	dq.AddAfter(first, 1*time.Second)
	dq.AddAfter(second, 500*time.Millisecond)

	fakeClock.Step(600 * time.Millisecond)

	err := checkAdded(dq, 1)
	if err != nil {
		t.Fatal(err)
	}
	x, ok := dq.Get()
	if !ok || x != second {
		t.Fatalf("expected %s, got %s", second, x)
	}

	fakeClock.Step(1 * time.Second)

	err = checkAdded(dq, 1)
	if err != nil {
		t.Fatal(err)
	}
	x, ok = dq.Get()
	if !ok || x != first {
		t.Fatalf("expected %s, got %s", first, x)
	}
}

// checkAdded checks whether the delay queue has the expected number of elements.
// The check action is supposed to be instantaneous.
// However, in order to reserve time for the internal synchronization of the delay queue itself,
// we perform the check in a polling manner and make it subject to a very short timeout(0.1s), the reserved time.
func checkAdded[T comparable](q DelayingQueue[T], len int) error {
	return wait.Poll(context.TODO(), func(ctx context.Context) (done bool, err error) {
		if q.Len() == len {
			return true, nil
		}
		return false, nil
	}, wait.WithInterval(1*time.Millisecond), wait.WithTimeout(100*time.Millisecond))
}
