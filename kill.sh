ps aux | grep demo | grep -v "grep" | awk '{print $2}' | xargs kill -9
