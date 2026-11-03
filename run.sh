#!/bin/bash
# Load secrets for ${VAR} references in the config (e.g. OPENROUTER_API_KEY).
if [ -f .env ]; then
	set -a
	. ./.env
	set +a
fi
go run ./cmd/lemon-chat "$@"
