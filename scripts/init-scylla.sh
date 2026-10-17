#!/bin/bash
set -euo pipefail

for f in /migrations/*.up.cql; do
  cqlsh scylla 9042 -f "$f"
done
