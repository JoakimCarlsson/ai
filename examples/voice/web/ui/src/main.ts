// main.ts — wire up the UI, audio capture/playback, and WebSocket client.

import { startMicCapture, PCMPlayer, type MicCapture } from "./audio";
import { VoiceWS, type VoiceEvent } from "./ws";

const startBtn = document.getElementById("start") as HTMLButtonElement;
const stopBtn = document.getElementById("stop") as HTMLButtonElement;
const statusEl = document.getElementById("status") as HTMLDivElement;
const conversationEl = document.getElementById("conversation") as HTMLDivElement;
const eventsEl = document.getElementById("events") as HTMLDivElement;

let mic: MicCapture | null = null;
let player: PCMPlayer | null = null;
let ws: VoiceWS | null = null;
let partialLine: HTMLSpanElement | null = null;
let assistantLine: HTMLSpanElement | null = null;

function setStatus(text: string): void {
  statusEl.textContent = text;
}

function appendConversation(text: string, cls: string): HTMLSpanElement {
  const line = document.createElement("div");
  const span = document.createElement("span");
  span.className = cls;
  span.textContent = text;
  line.appendChild(span);
  conversationEl.appendChild(line);
  conversationEl.scrollTop = conversationEl.scrollHeight;
  return span;
}

function appendEvent(text: string, cls: string): void {
  const line = document.createElement("div");
  line.className = cls;
  line.textContent = text;
  eventsEl.appendChild(line);
  eventsEl.scrollTop = eventsEl.scrollHeight;
}

function handleEvent(evt: VoiceEvent): void {
  appendEvent(JSON.stringify(evt), "info");

  switch (evt.type) {
    case "ready":
      setStatus("connected");
      break;
    case "user_transcript_partial":
      if (!partialLine) {
        partialLine = appendConversation("you (typing): ", "partial");
      }
      partialLine.textContent = `you (typing): ${evt.text ?? ""}`;
      break;
    case "user_transcript_final":
      if (partialLine) {
        partialLine.remove();
        partialLine = null;
      }
      appendConversation(`you: ${evt.text ?? ""}`, "final");
      assistantLine = null;
      break;
    case "assistant_delta":
      if (!assistantLine) {
        assistantLine = appendConversation("agent: ", "delta");
      }
      assistantLine.textContent =
        (assistantLine.textContent ?? "") + (evt.text ?? "");
      break;
    case "assistant_done":
      assistantLine = null;
      break;
    case "tool_call_start":
      appendConversation(
        `tool: ${evt.tool ?? ""}(${evt.toolArgs ?? ""})`,
        "tool",
      );
      break;
    case "tool_call_end":
      appendConversation(
        `tool result: ${evt.output ?? ""}`,
        evt.isError ? "error" : "tool",
      );
      break;
    case "error":
      appendConversation(`error: ${evt.error ?? ""}`, "error");
      break;
    case "conversation_end":
      setStatus("ended");
      break;
  }
}

async function start(): Promise<void> {
  startBtn.disabled = true;
  conversationEl.innerHTML = "";
  eventsEl.innerHTML = "";
  setStatus("connecting...");

  player = new PCMPlayer();
  ws = new VoiceWS();

  const url = `${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/ws`;
  ws.open(url, {
    onOpen: () => setStatus("opening mic..."),
    onClose: () => setStatus("disconnected"),
    onAudio: (frame) => player?.enqueue(frame),
    onEvent: handleEvent,
  });

  try {
    mic = await startMicCapture((frame) => ws?.sendAudio(frame));
    stopBtn.disabled = false;
  } catch (err) {
    setStatus(`mic error: ${(err as Error).message}`);
    await stop();
  }
}

async function stop(): Promise<void> {
  stopBtn.disabled = true;
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
  startBtn.disabled = false;
  setStatus("stopped");
}

startBtn.addEventListener("click", () => {
  void start();
});
stopBtn.addEventListener("click", () => {
  void stop();
});
