package auth

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
)

func TestFillFirstSelectorPick_Deterministic(t *testing.T) {
	t.Parallel()

	selector := &FillFirstSelector{}
	auths := []*Auth{
		{ID: "b"},
		{ID: "a"},
		{ID: "c"},
	}

	got, err := selector.Pick(context.Background(), "gemini", "", cliproxyexecutor.Options{}, auths)
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	if got == nil {
		t.Fatalf("Pick() auth = nil")
	}
	if got.ID != "a" {
		t.Fatalf("Pick() auth.ID = %q, want %q", got.ID, "a")
	}
}

func TestRoundRobinSelectorPick_CyclesDeterministic(t *testing.T) {
	t.Parallel()

	selector := &RoundRobinSelector{}
	auths := []*Auth{
		{ID: "b"},
		{ID: "a"},
		{ID: "c"},
	}

	want := []string{"a", "b", "c", "a", "b"}
	for i, id := range want {
		got, err := selector.Pick(context.Background(), "gemini", "", cliproxyexecutor.Options{}, auths)
		if err != nil {
			t.Fatalf("Pick() #%d error = %v", i, err)
		}
		if got == nil {
			t.Fatalf("Pick() #%d auth = nil", i)
		}
		if got.ID != id {
			t.Fatalf("Pick() #%d auth.ID = %q, want %q", i, got.ID, id)
		}
	}
}

func TestRoundRobinSelectorPick_PriorityBuckets(t *testing.T) {
	t.Parallel()

	selector := &RoundRobinSelector{}
	auths := []*Auth{
		{ID: "c", Attributes: map[string]string{"priority": "0"}},
		{ID: "a", Attributes: map[string]string{"priority": "10"}},
		{ID: "b", Attributes: map[string]string{"priority": "10"}},
	}

	want := []string{"a", "b", "a", "b"}
	for i, id := range want {
		got, err := selector.Pick(context.Background(), "mixed", "", cliproxyexecutor.Options{}, auths)
		if err != nil {
			t.Fatalf("Pick() #%d error = %v", i, err)
		}
		if got == nil {
			t.Fatalf("Pick() #%d auth = nil", i)
		}
		if got.ID != id {
			t.Fatalf("Pick() #%d auth.ID = %q, want %q", i, got.ID, id)
		}
		if got.ID == "c" {
			t.Fatalf("Pick() #%d unexpectedly selected lower priority auth", i)
		}
	}
}

func TestFillFirstSelectorPick_PriorityFallbackCooldown(t *testing.T) {
	t.Parallel()

	selector := &FillFirstSelector{}
	now := time.Now()
	model := "test-model"

	high := &Auth{
		ID:         "high",
		Attributes: map[string]string{"priority": "10"},
		ModelStates: map[string]*ModelState{
			model: {
				Status:         StatusActive,
				Unavailable:    true,
				NextRetryAfter: now.Add(30 * time.Minute),
				Quota: QuotaState{
					Exceeded: true,
				},
			},
		},
	}
	low := &Auth{ID: "low", Attributes: map[string]string{"priority": "0"}}

	got, err := selector.Pick(context.Background(), "mixed", model, cliproxyexecutor.Options{}, []*Auth{high, low})
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	if got == nil {
		t.Fatalf("Pick() auth = nil")
	}
	if got.ID != "low" {
		t.Fatalf("Pick() auth.ID = %q, want %q", got.ID, "low")
	}
}

func TestRoundRobinSelectorPick_Concurrent(t *testing.T) {
	selector := &RoundRobinSelector{}
	auths := []*Auth{
		{ID: "b"},
		{ID: "a"},
		{ID: "c"},
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	goroutines := 32
	iterations := 100
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				got, err := selector.Pick(context.Background(), "gemini", "", cliproxyexecutor.Options{}, auths)
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				if got == nil {
					select {
					case errCh <- errors.New("Pick() returned nil auth"):
					default:
					}
					return
				}
				if got.ID == "" {
					select {
					case errCh <- errors.New("Pick() returned auth with empty ID"):
					default:
					}
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()

	select {
	case err := <-errCh:
		t.Fatalf("concurrent Pick() error = %v", err)
	default:
	}
}

func TestWeightedSelectorPick_DefaultWeight(t *testing.T) {
	t.Parallel()

	seed := int64(1)
	selector := &WeightedSelector{rng: rand.New(rand.NewSource(seed))}
	auths := []*Auth{
		{ID: "b", Attributes: map[string]string{"weight": "3"}},
		{ID: "a"},
	}

	expectedRng := rand.New(rand.NewSource(seed))
	target := expectedRng.Int63n(4)
	expectedID := "b"
	if target < 1 {
		expectedID = "a"
	}

	got, err := selector.Pick(context.Background(), "gemini", "", cliproxyexecutor.Options{}, auths)
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	if got == nil {
		t.Fatalf("Pick() auth = nil")
	}
	if got.ID != expectedID {
		t.Fatalf("Pick() auth.ID = %q, want %q", got.ID, expectedID)
	}
}

func TestWeightedSelectorPick_IgnoresZeroWeight(t *testing.T) {
	t.Parallel()

	selector := &WeightedSelector{rng: rand.New(rand.NewSource(2))}
	auths := []*Auth{
		{ID: "a", Attributes: map[string]string{"weight": "0"}},
		{ID: "b", Attributes: map[string]string{"weight": "2"}},
	}

	for i := 0; i < 5; i++ {
		got, err := selector.Pick(context.Background(), "gemini", "", cliproxyexecutor.Options{}, auths)
		if err != nil {
			t.Fatalf("Pick() #%d error = %v", i, err)
		}
		if got == nil {
			t.Fatalf("Pick() #%d auth = nil", i)
		}
		if got.ID != "b" {
			t.Fatalf("Pick() #%d auth.ID = %q, want %q", i, got.ID, "b")
		}
	}
}

func TestWeightedSelectorPick_PriorityBuckets(t *testing.T) {
	t.Parallel()

	seed := int64(3)
	selector := &WeightedSelector{rng: rand.New(rand.NewSource(seed))}
	auths := []*Auth{
		{ID: "low", Attributes: map[string]string{"priority": "0", "weight": "100"}},
		{ID: "a", Attributes: map[string]string{"priority": "10", "weight": "1"}},
		{ID: "b", Attributes: map[string]string{"priority": "10", "weight": "3"}},
	}

	expectedRng := rand.New(rand.NewSource(seed))
	target := expectedRng.Int63n(4)
	expectedID := "b"
	if target < 1 {
		expectedID = "a"
	}

	got, err := selector.Pick(context.Background(), "mixed", "", cliproxyexecutor.Options{}, auths)
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	if got == nil {
		t.Fatalf("Pick() auth = nil")
	}
	if got.ID != expectedID {
		t.Fatalf("Pick() auth.ID = %q, want %q", got.ID, expectedID)
	}
	if got.ID == "low" {
		t.Fatalf("Pick() unexpectedly selected lower priority auth")
	}
}

func TestWeightedSelectorPick_AllWeightsZero(t *testing.T) {
	t.Parallel()

	selector := &WeightedSelector{rng: rand.New(rand.NewSource(4))}
	auths := []*Auth{
		{ID: "a", Attributes: map[string]string{"weight": "0"}},
		{ID: "b", Attributes: map[string]string{"weight": "-1"}},
		{ID: "c", Attributes: map[string]string{"weight": "nope"}},
	}

	got, err := selector.Pick(context.Background(), "gemini", "", cliproxyexecutor.Options{}, auths)
	if err == nil {
		t.Fatalf("Pick() error = nil")
	}
	if got != nil {
		t.Fatalf("Pick() auth = %v, want nil", got)
	}
	var authErr *Error
	if !errors.As(err, &authErr) {
		t.Fatalf("Pick() error type = %T, want *Error", err)
	}
	if authErr.Code != "auth_unavailable" {
		t.Fatalf("Pick() error code = %q, want %q", authErr.Code, "auth_unavailable")
	}
}

func TestWeightedSelectorPick_WeightOverflow(t *testing.T) {
	t.Parallel()

	selector := &WeightedSelector{rng: rand.New(rand.NewSource(5))}
	weightStr := strconv.FormatInt(math.MaxInt64, 10)
	auths := []*Auth{
		{ID: "a", Attributes: map[string]string{"weight": weightStr}},
		{ID: "b", Attributes: map[string]string{"weight": weightStr}},
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Pick() panic = %v", recovered)
		}
	}()

	got, err := selector.Pick(context.Background(), "gemini", "", cliproxyexecutor.Options{}, auths)
	if err == nil {
		t.Fatalf("Pick() error = nil")
	}
	if got != nil {
		t.Fatalf("Pick() auth = %v, want nil", got)
	}
	var authErr *Error
	if !errors.As(err, &authErr) {
		t.Fatalf("Pick() error type = %T, want *Error", err)
	}
	if authErr.Code != "auth_unavailable" {
		t.Fatalf("Pick() error code = %q, want %q", authErr.Code, "auth_unavailable")
	}
}
