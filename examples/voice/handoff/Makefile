.PHONY: dev

dev:
	@if not exist ui\node_modules ( cd ui && npm install )
	@start "voice-server" cmd /k "set GOWORK=off&& go run ."
	@cd ui && npm run dev
