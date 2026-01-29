package cliproxy

import (
	"testing"

	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

func TestNormalizeRoutingStrategy_DefaultWeighted(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"unknown",
		"random",
	}
	for _, input := range cases {
		if got := normalizeRoutingStrategy(input); got != routingStrategyWeighted {
			t.Fatalf("expected default strategy %q for %q, got %q", routingStrategyWeighted, input, got)
		}
	}
}

func TestNormalizeRoutingStrategyWithKnown(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		strategy string
		known    bool
	}{
		{name: "empty", input: "", strategy: routingStrategyWeighted, known: true},
		{name: "spaces", input: "   ", strategy: routingStrategyWeighted, known: true},
		{name: "fill-first", input: "ff", strategy: routingStrategyFillFirst, known: true},
		{name: "round-robin", input: "rr", strategy: routingStrategyRoundRobin, known: true},
		{name: "weighted", input: "weight", strategy: routingStrategyWeighted, known: true},
		{name: "unknown", input: "not-real", strategy: routingStrategyWeighted, known: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			strategy, known := normalizeRoutingStrategyWithKnown(tc.input)
			if strategy != tc.strategy {
				t.Fatalf("expected strategy %q for %q, got %q", tc.strategy, tc.input, strategy)
			}
			if known != tc.known {
				t.Fatalf("expected known=%t for %q, got %t", tc.known, tc.input, known)
			}
		})
	}
}

func TestNormalizeRoutingStrategy_Synonyms(t *testing.T) {
	cases := map[string][]string{
		routingStrategyFillFirst: {
			"fill-first",
			"fillfirst",
			"ff",
		},
		routingStrategyRoundRobin: {
			"round-robin",
			"roundrobin",
			"rr",
		},
		routingStrategyWeighted: {
			"weighted",
			"weight",
		},
	}

	for expected, inputs := range cases {
		for _, input := range inputs {
			if got := normalizeRoutingStrategy(input); got != expected {
				t.Fatalf("expected strategy %q for %q, got %q", expected, input, got)
			}
		}
	}
}

func TestSelectorForRoutingStrategy(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		match    func(coreauth.Selector) bool
		wantType string
	}{
		{
			name:  "fill-first",
			input: "fillfirst",
			match: func(sel coreauth.Selector) bool {
				_, ok := sel.(*coreauth.FillFirstSelector)
				return ok
			},
			wantType: "*auth.FillFirstSelector",
		},
		{
			name:  "round-robin",
			input: "rr",
			match: func(sel coreauth.Selector) bool {
				_, ok := sel.(*coreauth.RoundRobinSelector)
				return ok
			},
			wantType: "*auth.RoundRobinSelector",
		},
		{
			name:  "weighted",
			input: "weighted",
			match: func(sel coreauth.Selector) bool {
				_, ok := sel.(*coreauth.WeightedSelector)
				return ok
			},
			wantType: "*auth.WeightedSelector",
		},
		{
			name:  "default-weighted",
			input: "unknown",
			match: func(sel coreauth.Selector) bool {
				_, ok := sel.(*coreauth.WeightedSelector)
				return ok
			},
			wantType: "*auth.WeightedSelector",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			selector := selectorForRoutingStrategy(tc.input)
			if selector == nil {
				t.Fatalf("expected selector for %q, got nil", tc.input)
			}
			if !tc.match(selector) {
				t.Fatalf("expected selector type %s for %q, got %T", tc.wantType, tc.input, selector)
			}
		})
	}
}
