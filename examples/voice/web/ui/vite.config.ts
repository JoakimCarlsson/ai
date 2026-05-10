import { defineConfig } from "vite";

export default defineConfig({
  server: {
    port: 5173,
    proxy: {
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
        changeOrigin: true,
      },
    },
  },
});
