#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
if ! command -v cargo >/dev/null 2>&1; then
  echo "Rust/Cargo is required. Install stable Rust with https://rustup.rs/" >&2
  exit 1
fi
cargo build --release
exec ./target/release/bmie "$@"