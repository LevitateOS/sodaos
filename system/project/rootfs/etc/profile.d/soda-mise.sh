# A real shared installation, with per-user cache/config trust state.
export MISE_DATA_DIR=/opt/mise
export MISE_GLOBAL_CONFIG_FILE=/etc/mise/config.toml
export MISE_GLOBAL_CONFIG_ROOT=/etc/mise
case ":$PATH:" in *:/opt/mise/shims:*) ;; *) PATH="/opt/mise/shims:/usr/local/bin:$PATH";; esac
export PATH
