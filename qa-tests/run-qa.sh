#!/bin/bash
# PLM/SRM QA Test Runner
# Usage: ./run-qa.sh [suite] [--verbose]
#   suite: smoke | full (default: full)
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPORTS_DIR="$SCRIPT_DIR/reports"
AUTH_STATE="$SCRIPT_DIR/.auth-state.json"

mkdir -p "$REPORTS_DIR"

# Close any existing browser session
agent-browser close 2>/dev/null || true

# Generate fresh JWT auth state
echo "Generating auth token..."
node "$SCRIPT_DIR/gen-token.js" > "$AUTH_STATE"
export AGENT_BROWSER_STATE="$AUTH_STATE"

SUITE="${1:-full}"
VERBOSE=""
for arg in "$@"; do
  [ "$arg" = "--verbose" ] && VERBOSE="--verbose"
done

TIMESTAMP=$(date +%Y%m%d-%H%M%S)

case "$SUITE" in
  smoke)
    echo "Running PLM smoke test..."
    REPORT="$REPORTS_DIR/plm-smoke-${TIMESTAMP}.md"
    web-qa-bot run "$SCRIPT_DIR/plm-smoke.yaml" --output "$REPORT" $VERBOSE
    ;;
  full)
    echo "Running full PLM/SRM page test..."
    REPORT="$REPORTS_DIR/plm-full-${TIMESTAMP}.md"
    web-qa-bot run "$SCRIPT_DIR/plm-full.yaml" --output "$REPORT" $VERBOSE
    ;;
  *)
    echo "Unknown suite: $SUITE"
    echo "Usage: $0 [smoke|full] [--verbose]"
    exit 1
    ;;
esac

# Cleanup
agent-browser close 2>/dev/null || true
rm -f "$AUTH_STATE"

echo ""
echo "Report: $REPORT"
