set -eu

# ** SCRIPT START **
timer="go run $(go list -f '{{.Dir}}' myitcv.io/cmd/concsh)/_example/timer.go"
concsh -- $timer 1 --- $timer 2 --- $timer 3 --- $timer 4 --- $timer 5
