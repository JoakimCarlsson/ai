package model

// ProviderDeepgram identifies the Deepgram speech-to-text and text-to-speech
// provider.
const ProviderDeepgram Provider = "deepgram"

// Deepgram transcription model IDs.
const (
	DeepgramNova3       ID = "nova-3"
	DeepgramNova2       ID = "nova-2"
	DeepgramFluxEnglish ID = "flux-general-en"
	DeepgramFluxMulti   ID = "flux-general-multi"
)

// Deepgram Aura text-to-speech model/voice IDs. Deepgram's TTS model
// identifier encodes voice and language together (format:
// aura[-2]-<voice>-<lang>).
const (
	DeepgramAura2Thalia    ID = "aura-2-thalia-en"
	DeepgramAura2Andromeda ID = "aura-2-andromeda-en"
	DeepgramAura2Helena    ID = "aura-2-helena-en"
	DeepgramAura2Amalthea  ID = "aura-2-amalthea-en"
	DeepgramAura2Apollo    ID = "aura-2-apollo-en"
	DeepgramAura2Arcas     ID = "aura-2-arcas-en"
	DeepgramAura2Aries     ID = "aura-2-aries-en"
	DeepgramAura2Asteria   ID = "aura-2-asteria-en"
	DeepgramAura2Athena    ID = "aura-2-athena-en"
	DeepgramAura2Atlas     ID = "aura-2-atlas-en"
	DeepgramAura2Aurora    ID = "aura-2-aurora-en"
	DeepgramAura2Callista  ID = "aura-2-callista-en"
	DeepgramAura2Cora      ID = "aura-2-cora-en"
	DeepgramAura2Cordelia  ID = "aura-2-cordelia-en"
	DeepgramAura2Delia     ID = "aura-2-delia-en"
	DeepgramAura2Draco     ID = "aura-2-draco-en"
	DeepgramAura2Electra   ID = "aura-2-electra-en"
	DeepgramAura2Harmonia  ID = "aura-2-harmonia-en"
	DeepgramAura2Hera      ID = "aura-2-hera-en"
	DeepgramAura2Hermes    ID = "aura-2-hermes-en"
	DeepgramAura2Hyperion  ID = "aura-2-hyperion-en"
	DeepgramAura2Iris      ID = "aura-2-iris-en"
	DeepgramAura2Janus     ID = "aura-2-janus-en"
	DeepgramAura2Juno      ID = "aura-2-juno-en"
	DeepgramAura2Jupiter   ID = "aura-2-jupiter-en"
	DeepgramAura2Luna      ID = "aura-2-luna-en"
	DeepgramAura2Mars      ID = "aura-2-mars-en"
	DeepgramAura2Minerva   ID = "aura-2-minerva-en"
	DeepgramAura2Neptune   ID = "aura-2-neptune-en"
	DeepgramAura2Odysseus  ID = "aura-2-odysseus-en"
	DeepgramAura2Ophelia   ID = "aura-2-ophelia-en"
	DeepgramAura2Orion     ID = "aura-2-orion-en"
	DeepgramAura2Orpheus   ID = "aura-2-orpheus-en"
	DeepgramAura2Pandora   ID = "aura-2-pandora-en"
	DeepgramAura2Phoebe    ID = "aura-2-phoebe-en"
	DeepgramAura2Pluto     ID = "aura-2-pluto-en"
	DeepgramAura2Saturn    ID = "aura-2-saturn-en"
	DeepgramAura2Selene    ID = "aura-2-selene-en"
	DeepgramAura2Theia     ID = "aura-2-theia-en"
	DeepgramAura2Vesta     ID = "aura-2-vesta-en"
	DeepgramAura2Zeus      ID = "aura-2-zeus-en"
	DeepgramAuraAsteria    ID = "aura-asteria-en"
	DeepgramAuraLuna       ID = "aura-luna-en"
	DeepgramAuraStella     ID = "aura-stella-en"
	DeepgramAuraZeus       ID = "aura-zeus-en"
)

// DeepgramAudioModels maps Deepgram Aura TTS model IDs to their
// configurations. Each model identifies a specific voice + language; pick
// the variant that matches the desired voice. Aura-2 models offer lower
// latency (~90ms TTFB) and higher quality at a higher per-character cost.
var DeepgramAudioModels = map[ID]AudioModel{
	DeepgramAura2Amalthea: {
		ID:             DeepgramAura2Amalthea,
		Name:           "Deepgram Aura-2 Amalthea",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-amalthea-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Apollo: {
		ID:             DeepgramAura2Apollo,
		Name:           "Deepgram Aura-2 Apollo",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-apollo-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Arcas: {
		ID:             DeepgramAura2Arcas,
		Name:           "Deepgram Aura-2 Arcas",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-arcas-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Aries: {
		ID:             DeepgramAura2Aries,
		Name:           "Deepgram Aura-2 Aries",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-aries-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Asteria: {
		ID:             DeepgramAura2Asteria,
		Name:           "Deepgram Aura-2 Asteria",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-asteria-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Athena: {
		ID:             DeepgramAura2Athena,
		Name:           "Deepgram Aura-2 Athena",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-athena-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Atlas: {
		ID:             DeepgramAura2Atlas,
		Name:           "Deepgram Aura-2 Atlas",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-atlas-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Aurora: {
		ID:             DeepgramAura2Aurora,
		Name:           "Deepgram Aura-2 Aurora",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-aurora-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Callista: {
		ID:             DeepgramAura2Callista,
		Name:           "Deepgram Aura-2 Callista",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-callista-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Cora: {
		ID:             DeepgramAura2Cora,
		Name:           "Deepgram Aura-2 Cora",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-cora-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Cordelia: {
		ID:             DeepgramAura2Cordelia,
		Name:           "Deepgram Aura-2 Cordelia",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-cordelia-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Delia: {
		ID:             DeepgramAura2Delia,
		Name:           "Deepgram Aura-2 Delia",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-delia-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Draco: {
		ID:             DeepgramAura2Draco,
		Name:           "Deepgram Aura-2 Draco",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-draco-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Electra: {
		ID:             DeepgramAura2Electra,
		Name:           "Deepgram Aura-2 Electra",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-electra-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Harmonia: {
		ID:             DeepgramAura2Harmonia,
		Name:           "Deepgram Aura-2 Harmonia",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-harmonia-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Hera: {
		ID:             DeepgramAura2Hera,
		Name:           "Deepgram Aura-2 Hera",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-hera-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Hermes: {
		ID:             DeepgramAura2Hermes,
		Name:           "Deepgram Aura-2 Hermes",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-hermes-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Hyperion: {
		ID:             DeepgramAura2Hyperion,
		Name:           "Deepgram Aura-2 Hyperion",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-hyperion-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Iris: {
		ID:             DeepgramAura2Iris,
		Name:           "Deepgram Aura-2 Iris",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-iris-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Janus: {
		ID:             DeepgramAura2Janus,
		Name:           "Deepgram Aura-2 Janus",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-janus-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Juno: {
		ID:             DeepgramAura2Juno,
		Name:           "Deepgram Aura-2 Juno",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-juno-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Jupiter: {
		ID:             DeepgramAura2Jupiter,
		Name:           "Deepgram Aura-2 Jupiter",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-jupiter-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Luna: {
		ID:             DeepgramAura2Luna,
		Name:           "Deepgram Aura-2 Luna",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-luna-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Mars: {
		ID:             DeepgramAura2Mars,
		Name:           "Deepgram Aura-2 Mars",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-mars-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Minerva: {
		ID:             DeepgramAura2Minerva,
		Name:           "Deepgram Aura-2 Minerva",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-minerva-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Neptune: {
		ID:             DeepgramAura2Neptune,
		Name:           "Deepgram Aura-2 Neptune",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-neptune-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Odysseus: {
		ID:             DeepgramAura2Odysseus,
		Name:           "Deepgram Aura-2 Odysseus",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-odysseus-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Ophelia: {
		ID:             DeepgramAura2Ophelia,
		Name:           "Deepgram Aura-2 Ophelia",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-ophelia-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Orion: {
		ID:             DeepgramAura2Orion,
		Name:           "Deepgram Aura-2 Orion",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-orion-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Orpheus: {
		ID:             DeepgramAura2Orpheus,
		Name:           "Deepgram Aura-2 Orpheus",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-orpheus-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Pandora: {
		ID:             DeepgramAura2Pandora,
		Name:           "Deepgram Aura-2 Pandora",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-pandora-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Phoebe: {
		ID:             DeepgramAura2Phoebe,
		Name:           "Deepgram Aura-2 Phoebe",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-phoebe-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Pluto: {
		ID:             DeepgramAura2Pluto,
		Name:           "Deepgram Aura-2 Pluto",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-pluto-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Saturn: {
		ID:             DeepgramAura2Saturn,
		Name:           "Deepgram Aura-2 Saturn",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-saturn-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Selene: {
		ID:             DeepgramAura2Selene,
		Name:           "Deepgram Aura-2 Selene",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-selene-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Theia: {
		ID:             DeepgramAura2Theia,
		Name:           "Deepgram Aura-2 Theia",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-theia-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Vesta: {
		ID:             DeepgramAura2Vesta,
		Name:           "Deepgram Aura-2 Vesta",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-vesta-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Zeus: {
		ID:             DeepgramAura2Zeus,
		Name:           "Deepgram Aura-2 Zeus",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-zeus-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Thalia: {
		ID:             DeepgramAura2Thalia,
		Name:           "Deepgram Aura-2 Thalia",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-thalia-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Andromeda: {
		ID:             DeepgramAura2Andromeda,
		Name:           "Deepgram Aura-2 Andromeda",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-andromeda-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAura2Helena: {
		ID:             DeepgramAura2Helena,
		Name:           "Deepgram Aura-2 Helena",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-2-helena-en",
		CostPer1MChars: 30.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAuraAsteria: {
		ID:             DeepgramAuraAsteria,
		Name:           "Deepgram Aura Asteria",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-asteria-en",
		CostPer1MChars: 15.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAuraLuna: {
		ID:             DeepgramAuraLuna,
		Name:           "Deepgram Aura Luna",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-luna-en",
		CostPer1MChars: 15.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAuraStella: {
		ID:             DeepgramAuraStella,
		Name:           "Deepgram Aura Stella",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-stella-en",
		CostPer1MChars: 15.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
	DeepgramAuraZeus: {
		ID:             DeepgramAuraZeus,
		Name:           "Deepgram Aura Zeus",
		Provider:       ProviderDeepgram,
		APIModel:       "aura-zeus-en",
		CostPer1MChars: 15.00,
		SupportedFormats: []string{
			"mp3", "linear16", "mulaw", "alaw",
			"opus", "flac", "aac",
		},
		DefaultFormat:     "mp3",
		SupportsStreaming: true,
	},
}

// DeepgramTranscriptionModels maps Deepgram model IDs to their
// configurations. Both Nova-3 and Nova-2 support batch (HTTP POST) and
// streaming (WebSocket wss://api.deepgram.com/v1/listen). Streaming accepts
// linear16 PCM among other encodings; CostPer1MIn is the per-minute price.
var DeepgramTranscriptionModels = map[ID]TranscriptionModel{
	DeepgramFluxEnglish: {
		ID:            DeepgramFluxEnglish,
		Name:          "Deepgram Flux English",
		Provider:      ProviderDeepgram,
		APIModel:      "flux-general-en",
		CostPer1MIn:   0.0065,
		MaxFileSizeMB: 2000,
		SupportedFormats: []string{
			"mp3", "mp4", "wav", "flac",
			"ogg", "webm", "m4a",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    false,
		SupportsTranslation:    false,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
		},
	},
	DeepgramFluxMulti: {
		ID:            DeepgramFluxMulti,
		Name:          "Deepgram Flux Multilingual",
		Provider:      ProviderDeepgram,
		APIModel:      "flux-general-multi",
		CostPer1MIn:   0.0078,
		MaxFileSizeMB: 2000,
		SupportedFormats: []string{
			"mp3", "mp4", "wav", "flac",
			"ogg", "webm", "m4a",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    false,
		SupportsTranslation:    false,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json",
		},
	},
	DeepgramNova3: {
		ID:            DeepgramNova3,
		Name:          "Deepgram Nova 3",
		Provider:      ProviderDeepgram,
		APIModel:      "nova-3",
		CostPer1MIn:   0.0077,
		MaxFileSizeMB: 2000,
		SupportedFormats: []string{
			"mp3", "mp4", "wav", "flac",
			"ogg", "webm", "m4a",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsTranslation:    false,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json", "text", "srt", "vtt",
		},
	},
	DeepgramNova2: {
		ID:            DeepgramNova2,
		Name:          "Deepgram Nova 2",
		Provider:      ProviderDeepgram,
		APIModel:      "nova-2",
		CostPer1MIn:   0.0058,
		MaxFileSizeMB: 2000,
		SupportedFormats: []string{
			"mp3", "mp4", "wav", "flac",
			"ogg", "webm", "m4a",
		},
		SupportsTimestamps:     true,
		SupportsWordTimestamps: true,
		SupportsDiarization:    true,
		SupportsTranslation:    false,
		SupportsStreaming:      true,
		SupportedResponseFormats: []string{
			"json", "text", "srt", "vtt",
		},
	},
}
