// main.ts — UI state machine, audio capture/playback, WebSocket client.
//
// Layout: status pill, voice blob, chat bubble transcript, single mic button.
// A right-hand events panel mirrors every JSON event from the server.

import { startMicCapture, PCMPlayer, type MicCapture } from "./audio";
import { createBlob, type Blob, type BlobMode } from "./blob";
import { VoiceWS, type VoiceEvent } from "./ws";

type Status = "idle" | "connecting" | "listening" | "thinking" | "speaking";

const STATUS_LABEL: Record<Status, string> = {
  idle: "Idle",
  connecting: "Connecting",
  listening: "Listening",
  thinking: "Thinking",
  speaking: "Speaking",
};

const micBtn = document.getElementById("mic") as HTMLButtonElement;
const statusEl = document.getElementById("status") as HTMLDivElement;
const transcriptEl = document.getElementById("transcript") as HTMLDivElement;
const eventsEl = document.getElementById("events") as HTMLDivElement;
const blobHost = document.getElementById("blob") as HTMLDivElement;
const blob: Blob = createBlob(blobHost);

let mic: MicCapture | null = null;
let player: PCMPlayer | null = null;
let ws: VoiceWS | null = null;
let drainTimer: number | null = null;
let userPartialEl: HTMLDivElement | null = null;
let assistantTurnEl: HTMLDivElement | null = null;

setStatus("idle");

function setStatus(s: Status): void {
  statusEl.dataset.tone = s;
  statusEl.querySelector(".status-label")!.textContent = STATUS_LABEL[s];
  const blobMode: BlobMode =
    s === "speaking" ? "assistant" : s === "listening" || s === "thinking" ? "user" : "idle";
  blob.setMode(blobMode);
  blob.setGetLevel(() => {
    if (s === "speaking") return player?.level() ?? 0;
    if (s === "listening" || s === "thinking") return mic?.level() ?? 0;
    return 0;
  });
  micBtn.dataset.active = s === "idle" ? "false" : "true";
  micBtn.setAttribute("aria-label", s === "idle" ? "Start" : "Stop");
}

function appendEvent(evt: VoiceEvent): void {
  const line = document.createElement("div");
  line.className = "event";
  line.dataset.type = evt.type;
  line.textContent = JSON.stringify(evt);
  eventsEl.appendChild(line);
  eventsEl.scrollTop = eventsEl.scrollHeight;
}

function addUserBubble(text: string, partial: boolean): HTMLDivElement {
  const wrap = document.createElement("div");
  wrap.className = "bubble user";
  if (partial) wrap.classList.add("partial");
  wrap.textContent = text;
  transcriptEl.appendChild(wrap);
  transcriptEl.scrollTop = transcriptEl.scrollHeight;
  return wrap;
}

function addAssistantBubble(): HTMLDivElement {
  const wrap = document.createElement("div");
  wrap.className = "bubble assistant partial";
  wrap.textContent = "";
  transcriptEl.appendChild(wrap);
  transcriptEl.scrollTop = transcriptEl.scrollHeight;
  return wrap;
}

function cancelDrain(): void {
  if (drainTimer !== null) {
    clearTimeout(drainTimer);
    drainTimer = null;
  }
}

function scheduleDrainAfterTTS(): void {
  const tick = (): void => {
    drainTimer = null;
    const remaining = player?.remainingTime() ?? 0;
    if (remaining <= 0) {
      setStatus("listening");
      return;
    }
    drainTimer = window.setTimeout(tick, Math.min(200, remaining * 1000 + 30));
  };
  tick();
}

function handleEvent(evt: VoiceEvent): void {
  appendEvent(evt);

  switch (evt.type) {
    case "ready":
      setStatus("listening");
      break;
    case "user_transcript_partial":
      if (!userPartialEl) {
        userPartialEl = addUserBubble(evt.text ?? "", true);
      } else {
        userPartialEl.textContent = evt.text ?? "";
      }
      setStatus("listening");
      break;
    case "user_transcript_final":
      if (userPartialEl) {
        userPartialEl.textContent = evt.text ?? "";
        userPartialEl.classList.remove("partial");
        userPartialEl = null;
      } else if ((evt.text ?? "").trim()) {
        addUserBubble(evt.text ?? "", false);
      }
      assistantTurnEl = null;
      setStatus("thinking");
      break;
    case "assistant_delta":
      if (!assistantTurnEl) {
        assistantTurnEl = addAssistantBubble();
      }
      assistantTurnEl.textContent =
        (assistantTurnEl.textContent ?? "") + (evt.text ?? "");
      transcriptEl.scrollTop = transcriptEl.scrollHeight;
      break;
    case "assistant_done":
      if (assistantTurnEl) {
        assistantTurnEl.classList.remove("partial");
      }
      break;
    case "agent_interrupted":
      void player?.flush();
      if (assistantTurnEl) {
        assistantTurnEl.classList.remove("partial");
        assistantTurnEl.textContent =
          (assistantTurnEl.textContent ?? "") + " [interrupted]";
      } else if (evt.text) {
        const bubble = addAssistantBubble();
        bubble.textContent = `${evt.text} [interrupted]`;
        bubble.classList.remove("partial");
      }
      assistantTurnEl = null;
      break;
    case "tool_call_start": {
      const line = document.createElement("div");
      line.className = "bubble tool";
      line.textContent = `→ ${evt.tool ?? ""}(${evt.toolArgs ?? ""})`;
      transcriptEl.appendChild(line);
      transcriptEl.scrollTop = transcriptEl.scrollHeight;
      break;
    }
    case "tool_call_end": {
      const line = document.createElement("div");
      line.className = "bubble tool";
      if (evt.isError) line.classList.add("error");
      line.textContent = `← ${evt.tool ?? ""}: ${evt.output ?? ""}`;
      transcriptEl.appendChild(line);
      transcriptEl.scrollTop = transcriptEl.scrollHeight;
      break;
    }
    case "tts_started":
      cancelDrain();
      setStatus("speaking");
      break;
    case "tts_ended":
      scheduleDrainAfterTTS();
      break;
    case "error": {
      const line = document.createElement("div");
      line.className = "bubble error";
      line.textContent = `error: ${evt.error ?? ""}`;
      transcriptEl.appendChild(line);
      break;
    }
    case "conversation_end":
      setStatus("idle");
      break;
  }
}

async function start(): Promise<void> {
  micBtn.disabled = true;
  setStatus("connecting");
  transcriptEl.replaceChildren();
  eventsEl.replaceChildren();
  userPartialEl = null;
  assistantTurnEl = null;

  player = new PCMPlayer();
  ws = new VoiceWS();

  const url = `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/ws`;
  ws.open(url, {
    onOpen: () => setStatus("listening"),
    onClose: () => setStatus("idle"),
    onAudio: (frame) => player?.enqueue(frame),
    onEvent: handleEvent,
  });

  try {
    mic = await startMicCapture((frame) => ws?.sendAudio(frame));
  } catch (err) {
    handleEvent({ type: "error", error: (err as Error).message });
    await stop();
    return;
  }
  micBtn.disabled = false;
}

async function stop(): Promise<void> {
  micBtn.disabled = true;
  cancelDrain();
  if (mic) {
    await mic.stop();
    mic = null;
  }
  if (ws) {
    ws.close();
    ws = null;
  }
  if (player) {
    await player.close();
    player = null;
  }
  setStatus("idle");
  micBtn.disabled = false;
}

micBtn.addEventListener("click", () => {
  if (micBtn.dataset.active === "true") {
    void stop();
  } else {
    void start();
  }
});
