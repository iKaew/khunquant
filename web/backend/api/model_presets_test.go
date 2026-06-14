package api

import "testing"

func TestDefaultModelForProvider(t *testing.T) {
	cases := []struct {
		provider string
		want     string
	}{
		{oauthProviderOpenAI, "openai/gpt-5.5"},
		{oauthProviderAnthropic, "anthropic/claude-sonnet-4.6"},
		{oauthProviderGoogleAntigravity, "antigravity/gemini-3-flash"},
		{oauthProviderGoogleGemini, "gemini-code-assist/gemini-2.5-flash"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			got := defaultModelForProvider(tc.provider)
			if got != tc.want {
				t.Errorf("defaultModelForProvider(%q) = %q, want %q", tc.provider, got, tc.want)
			}
		})
	}
}

// TestDefaultIsListedInPresets ensures each provider's default model ID appears somewhere in its
// preset list. Anthropic's default (sonnet) is intentionally not the first preset (fable-5 is
// listed first as the most capable), but it must still appear so users can select it via the UI.
func TestDefaultIsListedInPresets(t *testing.T) {
	providers := []string{
		oauthProviderOpenAI,
		oauthProviderAnthropic,
		oauthProviderGoogleAntigravity,
	}
	for _, provider := range providers {
		presets, ok := oauthProviderModelPresets[provider]
		if !ok || len(presets) == 0 {
			t.Errorf("provider %q has no presets", provider)
			continue
		}
		want := defaultModelForProvider(provider)
		found := false
		for _, p := range presets {
			if p.ModelID == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("provider %q: defaultModelForProvider returns %q but that model is not in the presets list", provider, want)
		}
	}
}

func TestPresetsAllHaveNonEmptyLabelAndModelID(t *testing.T) {
	for provider, presets := range oauthProviderModelPresets {
		for i, p := range presets {
			if p.Label == "" {
				t.Errorf("provider %q preset[%d]: Label is empty", provider, i)
			}
			if p.ModelID == "" {
				t.Errorf("provider %q preset[%d]: ModelID is empty", provider, i)
			}
		}
	}
}
