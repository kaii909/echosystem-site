#!/bin/bash
# ============================================
# echosystem site development server
# templ watch + go run
# press 'q' to stop, 'r' to restart
# ============================================

set -e

echo "========================================="
echo "  echosystem development server"
echo "========================================="
echo ""
echo "[IMPORTANT] if the program asks for input, run 'go run .' before using dev.sh"
echo "starting templ generator with proxy..."
echo "press 'q' to stop, 'r' to restart the server."
echo ""

# start server
start_server() {
	go tool templ generate --watch --proxy="http://localhost:8080" --cmd="go run ." --open-browser=false &
	TEMPL_PID=$!
}

# initial start
start_server

# cleanup
cleanup() {
	echo ""
	echo "stopping development server..."
	kill $TEMPL_PID 2>/dev/null
	wait $TEMPL_PID 2>/dev/null
	echo "server stopped."
	exit 0
}

trap cleanup EXIT INT TERM

# input loop
while true; do
	read -r input
	if [[ "$input" == "q" || "$input" == "Q" ]]; then
		break
	elif [[ "$input" == "r" || "$input" == "R" ]]; then
		echo ""
		echo "restarting development server..."
		kill $TEMPL_PID 2>/dev/null
		wait $TEMPL_PID 2>/dev/null
		echo "starting templ generator with proxy..."
		start_server
	fi
done
