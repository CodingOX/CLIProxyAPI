package cliproxy

import (
	"strings"

	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

const (
	routingStrategyWeighted   = "weighted"
	routingStrategyRoundRobin = "round-robin"
	routingStrategyFillFirst  = "fill-first"
)

func normalizeRoutingStrategyWithKnown(strategy string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(strategy))
	if normalized == "" {
		return routingStrategyWeighted, true
	}
	switch normalized {
	case routingStrategyFillFirst, "fillfirst", "ff":
		return routingStrategyFillFirst, true
	case routingStrategyRoundRobin, "roundrobin", "rr":
		return routingStrategyRoundRobin, true
	case routingStrategyWeighted, "weight":
		return routingStrategyWeighted, true
	default:
		return routingStrategyWeighted, false
	}
}

func normalizeRoutingStrategy(strategy string) string {
	normalized, _ := normalizeRoutingStrategyWithKnown(strategy)
	return normalized
}

func selectorForRoutingStrategy(strategy string) coreauth.Selector {
	switch normalizeRoutingStrategy(strategy) {
	case routingStrategyFillFirst:
		return &coreauth.FillFirstSelector{}
	case routingStrategyRoundRobin:
		return &coreauth.RoundRobinSelector{}
	default:
		return &coreauth.WeightedSelector{}
	}
}
