_fzf_complete_ollama() {
      local -a tokens
      tokens=(${(z)1})
      case ${tokens[2]} in
        pull|run)
            # Use ollamasearch for ollama pull with wrapper
            _fzf_complete \
                --prompt="ollamasearch> " \
                --bind 'start:reload:ollamasearch {q}' \
                --bind 'change:reload:ollamasearch {q}' \
                --ansi \
                --disabled \
                -- "$@" < <(ollamasearch "${tokens[@]:2}")
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
