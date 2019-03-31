#!/usr/bin/env bash
set -e

cp /pipehub.hcl cmd/pipehub/
make build
mkdir -p output
cp cmd/pipehub/pipehub output