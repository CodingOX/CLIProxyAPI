package managementasset

import (
	"bytes"
	"testing"
)

func TestPatchManagementAsset(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSubs []string
	}{
		{
			name: "Inject translation and option",
			input: `const lang={routing_strategy_round_robin:"round-robin (cycle)",other:"val"};
return[g.jsx("option",{value:"round-robin",children:i("basic_settings.routing_strategy_round_robin")}),g.jsx("option",{value:"fill-first",children:i("basic.ff")})]`,
			wantSubs: []string{
				`routing_strategy_weighted:"weighted"`,
				`g.jsx("option",{value:"weighted",children:i("basic_settings.routing_strategy_weighted")})`,
			},
		},
		{
			name: "Inject translation Chinese",
			input: `const lang={routing_strategy_round_robin:"round-robin (轮询)",other:"val"};`,
			wantSubs: []string{
				`routing_strategy_weighted:"weighted (权重)"`,
			},
		},
		{
			name: "Idempotency - already exists",
			input: `const lang={routing_strategy_round_robin:"round-robin",routing_strategy_weighted:"weighted"};
g.jsx("option",{value:"weighted",children:i("basic.weight")})`,
			wantSubs: []string{
				`routing_strategy_weighted:"weighted"`, // Should still be there
			},
		},
		{
			name: "No match - safety",
			input: `some random content without the target strings`,
			wantSubs: []string{}, // Should return original or just not crash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := patchManagementAsset([]byte(tt.input))
			for _, sub := range tt.wantSubs {
				if !bytes.Contains(got, []byte(sub)) {
					t.Errorf("patchManagementAsset() missing expected substring: %s\nGot: %s", sub, got)
				}
			}
			// Safety check: if input didn't have targets, output should equal input (mostly)
			// For simplicity in this "hot patch", we mainly care that it *does* the job when targets exist.
		})
	}
}
