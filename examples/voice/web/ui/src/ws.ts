// ws.ts — minimal WebSocket client for the voice agent.
//
// Sends mic PCM frames as binary messages and receives:
//   - binary frames: TTS audio (16 kHz mono PCM int16)
//   - text frames: JSON-encoded events from the voice agent

export type VoiceEventType =
  | "ready"
  | "user_transcript_partial"
  | "user_transcript_final"
  | "assistant_delta"
  | "assistant_done"
  | "tool_call_start"
  | "tool_call_end"
  | "tts_started"
  | "tts_ended"
  | "conversation_end"
  | "error";

export interface VoiceEvent {
  type: VoiceEventType;
  text?: string;
  tool?: string;
  toolId?: string;
  toolArgs?: string;
  output?: string;
  isError?: boolean;
  error?: string;
}

export interface VoiceWSHandlers {
  onAudio: (frame: ArrayBuffer) => void;
  onEvent: (evt: VoiceEvent) => void;
  onOpen?: () => void;
  onClose?: (ev: CloseEvent) => void;
}

export class VoiceWS {
  private ws: WebSocket | null = null;

  open(url: string, handlers: VoiceWSHandlers): void {
    const ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";

    ws.addEventListener("open", () => handlers.onOpen?.());
    ws.addEventListener("close", (ev) => handlers.onClose?.(ev));
    ws.addEventListener("message", (event) => {
      if (typeof event.data === "string") {
        try {
          const evt = JSON.parse(event.data) as VoiceEvent;
          handlers.onEvent(evt);
        } catch {
          // ignore malformed text frames
        }
        return;
      }
      if (event.data instanceof ArrayBuffer) {
        handlers.onAudio(event.data);
      }
    });

    this.ws = ws;
  }

  sendAudio(frame: ArrayBuffer): void {
    const ws = this.ws;
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(frame);
  }

  close(): void {
    this.ws?.close();
    this.ws = null;
  }
}
