package llm

import "testing"

func TestParseModel(t *testing.T) {
	tests := []struct {
		input     string
		wantProv  string
		wantModel string
		wantErr   bool
	}{
		{"anthropic:claude-haiku-4-5", "anthropic", "claude-haiku-4-5", false},
		{"openai:gpt-4o", "openai", "gpt-4o", false},
		{"openai:meta-llama/Llama-3-8B", "openai", "meta-llama/Llama-3-8B", false},
		{"fireworks:accounts/fireworks/models/deepseek-v3p2", "fireworks", "accounts/fireworks/models/deepseek-v3p2", false},
		{"ollama:llama3", "ollama", "llama3", false},
		{"ollama:qwen2.5-coder:7b", "ollama", "qwen2.5-coder:7b", false},
		{"claude-haiku-4-5", "", "", true},
		{"", "", "", true},
	}
	for _, tt := range tests {
		prov, model, err := ParseModel(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseModel(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			continue
		}
		if prov != tt.wantProv {
			t.Errorf("ParseModel(%q) provider = %q, want %q", tt.input, prov, tt.wantProv)
		}
		if model != tt.wantModel {
			t.Errorf("ParseModel(%q) model = %q, want %q", tt.input, model, tt.wantModel)
		}
	}
}
