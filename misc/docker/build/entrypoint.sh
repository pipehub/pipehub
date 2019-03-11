#!/usr/bin/env bash
set -e

cp /pipehub.hcl cmd/pipehub/
make generate
mkdir -p output
cp cmd/pipehub/pipehub output