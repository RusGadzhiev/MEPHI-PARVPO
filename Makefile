.PHONY: start
start:
	pg_pass="pass" docker compose up --build

.DEFAULT_GOAL := start