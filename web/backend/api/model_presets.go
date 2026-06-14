package api

// ─────────────────────────────────────────────────────────────────────────────
// Credential-card model presets
//
// UPDATE THIS FILE when new models are released (frontier models change ~monthly).
// Each provider block is independent — just edit labels and model IDs.
//
// ModelID format: {provider-prefix}/{api-model-id}
//   openai/       → OpenAI API (Codex CLI uses ChatGPT session auth)
//   anthropic/    → Anthropic API
//   antigravity/  → Google Antigravity (cloudcode-pa.googleapis.com)
//
// Last verified: 2026-06-14
// ─────────────────────────────────────────────────────────────────────────────

// oauthProviderModelPresets defines the clickable model chips shown on each
// credential card. Order matters — first entry is shown leftmost/topmost.
//
// Sources checked when last updated:
//
//	OpenAI:      developers.openai.com/codex/models
//	Anthropic:   platform.claude.com/docs/en/about-claude/models/overview
//	Antigravity: github.com/NoeFabris/opencode-antigravity-auth (PR #574)
var oauthProviderModelPresets = map[string][]oauthModelPreset{
	// ── OpenAI ────────────────────────────────────────────────────────────────
	// gpt-5.5  = current default for ChatGPT-authenticated Codex CLI sessions
	// gpt-5.4  = stable fallback
	// gpt-5.4-mini = lighter tasks, subagents, interactive edits
	oauthProviderOpenAI: {
		{Label: "GPT-5.5", ModelID: "openai/gpt-5.5"},
		{Label: "GPT-5.4", ModelID: "openai/gpt-5.4"},
		{Label: "GPT-5.4 Mini", ModelID: "openai/gpt-5.4-mini"},
	},

	// ── Anthropic ─────────────────────────────────────────────────────────────
	// claude-fable-5   = Mythos class, released 2026-06-09 (top tier, $10/$50/MTok)
	// claude-sonnet-4.6 = recommended production default ($3/$15/MTok)
	// claude-opus-4.8  = heavy reasoning ($5/$25/MTok)
	// claude-haiku-4.5 = low-latency, low-cost ($1/$5/MTok)
	oauthProviderAnthropic: {
		{Label: "Claude Fable 5", ModelID: "anthropic/claude-fable-5"},
		{Label: "Claude Sonnet 4.6", ModelID: "anthropic/claude-sonnet-4.6"},
		{Label: "Claude Opus 4.8", ModelID: "anthropic/claude-opus-4.8"},
		{Label: "Claude Haiku 4.5", ModelID: "anthropic/claude-haiku-4.5"},
	},

	// ── Google Antigravity ────────────────────────────────────────────────────
	// Model IDs come from the Cloud Code Assist v1internal API (Antigravity quota).
	// Source: github.com/NoeFabris/opencode-antigravity-auth README (2026-06-14)
	// The provider strips the "antigravity/" prefix before calling the API.
	// Note: gemini-3-flash-preview is a Gemini CLI quota model (different API) — NOT valid here.
	// Note: gemini-3.1-pro uses dot notation (rollout-dependent; may be unavailable on some accounts).
	// Claude models (claude-sonnet-4-6, claude-opus-4-6-thinking) also route through Antigravity quota.
	oauthProviderGoogleAntigravity: {
		{Label: "Gemini 3 Flash", ModelID: "antigravity/gemini-3-flash"},
		{Label: "Gemini 3 Pro", ModelID: "antigravity/gemini-3-pro"},
		{Label: "Gemini 3.1 Pro", ModelID: "antigravity/gemini-3.1-pro"},
		{Label: "Claude Sonnet 4.6", ModelID: "antigravity/claude-sonnet-4-6"},
		{Label: "Claude Opus 4.6 Thinking", ModelID: "antigravity/claude-opus-4-6-thinking"},
	},
}

// defaultModelForProvider returns the model ID to use when a provider is first
// connected and no existing model_list entry exists for it.
// Keep in sync with the first entry of each provider block above.
func defaultModelForProvider(provider string) string {
	switch provider {
	case oauthProviderOpenAI:
		return "openai/gpt-5.5"
	case oauthProviderAnthropic:
		return "anthropic/claude-sonnet-4.6" // sonnet is recommended production default
	case oauthProviderGoogleAntigravity:
		return "antigravity/gemini-3-flash"
	case oauthProviderGoogleGemini:
		return "gemini-code-assist/gemini-2.5-flash"
	default:
		return ""
	}
}
