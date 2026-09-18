# Dev commands for the divoom dashboard daemon. No build/push/deploy targets
# here — this fork runs via `docker compose build && docker compose up -d`
# directly on the host with the frame USB-attached (see README.md), not the
# upstream GHCR + Portainer pipeline that used to live in this file.

ENV_FILE := .env

.PHONY: test vet lint fmt run probe render-out push-frame

test:
	go test ./...

vet:
	go vet ./...

# Optional — only if golangci-lint is on PATH.
lint:
	@command -v golangci-lint >/dev/null && golangci-lint run \
	    || echo "golangci-lint not installed; skipping"

fmt:
	gofmt -w .

# withenv sources .env into the recipe shell so locally-running
# subcommands see HA_URL / HA_TOKEN / DIVOOM_FRAME_IP the same way the
# container does.
withenv = set -a; [ -f $(ENV_FILE) ] && . ./$(ENV_FILE); set +a;

# Run the daemon locally against the configured frame.
run:
	$(withenv) go run ./cmd/divoom serve

probe:
	$(withenv) go run ./cmd/divoom probe

# Render the scene background(s) to ./dist/scenes/ for inspection.
render-out:
	$(withenv) go run ./cmd/divoom render

# Push scene backgrounds + custom fonts to the frame via adb. Runs against
# whatever device adb sees on this host. After any scene/layout change or a
# factory reset, run this from wherever the frame is currently USB-attached.
push-frame:
	$(withenv) go run ./cmd/divoom push
