#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

DAEMON_PID=""
FRONTEND_PID=""

echo "========================================="
echo "API Watcher - Starting Services"
echo "========================================="
echo ""

# Cleanup function for graceful shutdown
cleanup() {
    echo ""
    echo "Shutting down services..."
    if [ -n "$DAEMON_PID" ]; then
        kill $DAEMON_PID 2>/dev/null || true
        wait $DAEMON_PID 2>/dev/null || true
    fi
    if [ -n "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
        wait $FRONTEND_PID 2>/dev/null || true
    fi
    echo "Services stopped."
}

trap cleanup EXIT

# Error handler - kills all processes and exits
error_exit() {
    echo ""
    echo "❌ ERROR: $1"
    echo "Killing all services..."
    cleanup
    exit 1
}

# Build daemon
echo "Building daemon..."
if ! go build -o apiwatcher-daemon ./cmd/apiwatcher-daemon/; then
    error_exit "Failed to build daemon"
fi
echo "✓ Daemon built"
echo ""

# Start daemon in background
echo "Starting daemon..."
./apiwatcher-daemon &
DAEMON_PID=$!
echo "✓ Daemon started (PID: $DAEMON_PID)"
echo ""

# Wait for daemon to start and verify it's running
sleep 2
if ! kill -0 $DAEMON_PID 2>/dev/null; then
    error_exit "Daemon process exited unexpectedly"
fi

# Check if daemon is listening on port 9876
if ! lsof -i :9876 &>/dev/null; then
    error_exit "Daemon is not listening on port 9876. Is it running correctly?"
fi
echo "✓ Daemon verified running on port 9876"
echo ""

# Start frontend dev server
echo "Starting frontend dev server..."
cd frontend
npm run serve &
FRONTEND_PID=$!
cd ..
echo "✓ Frontend dev server started (PID: $FRONTEND_PID)"
echo ""

# Wait a moment for frontend to start
sleep 2
if ! kill -0 $FRONTEND_PID 2>/dev/null; then
    error_exit "Frontend dev server exited unexpectedly"
fi
echo "✓ Frontend dev server verified running"
echo ""

# Verify daemon is still running
if ! kill -0 $DAEMON_PID 2>/dev/null; then
    error_exit "Daemon crashed while starting frontend"
fi

echo "========================================="
echo "✓ All services started successfully!"
echo "========================================="
echo ""

# Start Wails
echo "Starting Wails..."
/home/andy007/go/bin/wails serve
if [ $? -ne 0 ]; then
    error_exit "Wails failed to start"
fi

