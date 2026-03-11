#!/bin/bash
# Install goatlas post-commit hook into a git repository.
# Usage: goatlas-hook install <repo-path>
#        goatlas-hook uninstall <repo-path>

set -e

install_hook() {
    local repo="$1"
    local hook_file="$repo/.git/hooks/post-commit"

    if [ ! -d "$repo/.git" ]; then
        echo "Error: $repo is not a git repository"
        exit 1
    fi

    cat > "$hook_file" << 'EOF'
#!/bin/bash
# GoAtlas auto-refresh: re-index + rebuild graph after each commit
REPO_PATH="$(git rev-parse --toplevel)"
(goatlas index "$REPO_PATH" && goatlas build-graph) &>/dev/null &
disown
EOF
    chmod +x "$hook_file"
    echo "✅ GoAtlas hook installed in $repo"
}

uninstall_hook() {
    local repo="$1"
    local hook_file="$repo/.git/hooks/post-commit"
    if [ -f "$hook_file" ]; then
        rm "$hook_file"
        echo "✅ GoAtlas hook removed from $repo"
    else
        echo "No hook found in $repo"
    fi
}

case "$1" in
    install)   install_hook "$2" ;;
    uninstall) uninstall_hook "$2" ;;
    *)         echo "Usage: goatlas-hook install|uninstall <repo-path>" ;;
esac
