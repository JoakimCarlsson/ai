// Package voice provides a voice-first conversational agent that runs an STT
// → LLM → TTS pipeline over a duplex audio transport.
//
// The agent is provider-agnostic: pass any [stt.SpeechToText] and any
// [tts.Generation] implementation to [New]. If the TTS client also implements
// [tts.StreamingTextProvider], the pipeline streams LLM tokens directly into
// TTS for end-to-end concurrent text-to-audio. Otherwise it buffers the LLM
// output at sentence boundaries and falls back to single-shot
// [tts.Generation.StreamAudio] per sentence.
//
// Tool calls are executed inline within an assistant turn, with a configurable
// iteration cap.
//
// Example:
//
//	v := voice.New(llmClient, sttClient, ttsClient,
//	    voice.WithSystemPrompt("You are a helpful assistant."),
//	    voice.WithInitialMessage("Hi, how can I help?"),
//	    voice.WithTools(myTool),
//	)
//	conv, err := v.StartConversation(ctx, transport)
//	if err != nil { /* ... */ }
//	for evt := range conv.Events() {
//	    // observe transcripts, deltas, tool calls, ...
//	}
//	if err := conv.Wait(); err != nil { /* ... */ }
package voice
