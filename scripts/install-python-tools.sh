#!/usr/bin/env bash

set -euo pipefail

python3 -m pip install --require-hashes -r requirements-lock.txt
asdf reshim
