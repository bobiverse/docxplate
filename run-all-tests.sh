#!/usr/bin/env bash

# Make sure to fail on any error
set -e

echo -e "\n\n--- Lint -----------------------------------------------------------"
RESULT="$(golint)"
if [[ ! -z "$RESULT" ]]; then echo "$RESULT"; exit 1; fi

echo -e "\n\n--- Test -----------------------------------------------------------"
go test -v

echo -e "\n\n--- Race -----------------------------------------------------------"
go test -v -race

echo -e "\n\n--- Coverage -------------------------------------------------------"
go test ./... -cover

echo -e "\n\n"
