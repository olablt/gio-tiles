# tmux new -s gio-tiles
tmux send-keys -t gio-tiles.0 C-c;
tmux send-keys -t gio-tiles.0 C-l;
tmux send-keys -t gio-tiles.0 "tmux clear-history" ENTER

# tmux send-keys -t gio-tiles.0 "go run ." ENTER
tmux send-keys -t gio-tiles.0 "go run ./apps/hello/" ENTER
