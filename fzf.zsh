_fzf_complete_ollama() {
      local -a tokens
      tokens=(${(z)1})
      case ${tokens[-1]} in
        pull|run)
            # Use blake.io/ollamasearch for ollama pull
            _fzf_complete \
                --prompt="ollamasearch> " \
                --bind 'start:reload:ollamasearch ""' \
                --bind 'change:reload:ollamasearch {q}' \
                --ansi \
                --disabled \
                -- "$@" < <(ollamasearch "$@")
            ;;
        *)
            _fzf_complete \
                --prompt="ollama> " \
                --ansi \
                -- "$@" < <(ollama ls)
    esac
}

# Extract first column (model name) for selection
_fzf_complete_ollama_post() {
    awk '{print $1}'
}
