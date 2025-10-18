# bash completion for sg-fw
# put in /etc/bash_completion.d/
# you'll need `bash-completion` installed for your distro

_sg_fw_fetch_serial_devices() {
    local devices=()
    if [[ -d /dev/serial/by-id ]]; then
        for dev in /dev/serial/by-id/*; do
            [[ -e "$dev" ]] && devices+=("$dev")
        done
    fi
    echo "${devices[@]}"
}

_sg_fw() {
    local cur prev words cword
    if type _get_comp_words_by_ref >/dev/null 2>&1; then
        _get_comp_words_by_ref -n : cur prev words cword
    else
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
    fi

    local commands="serial make flash help"
    local repos="robot basestation"

    if [[ $cword -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
        return 0
    fi

    case "${words[1]}" in
        make|flash)
            if [[ $cword -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$repos" -- "$cur") )
                return 0
            fi
            ;;
        serial)
            if [[ $cword -eq 2 ]]; then
                local devices
                devices=$(_sg_fw_fetch_serial_devices)
                if [[ -z "$devices" ]]; then
                    echo "No serial devices found" >&2
                    return 0
                fi
                COMPREPLY=( $(compgen -W "$devices" -- "$cur") )
                return 0
            fi
            ;;
    esac

    if [[ ${words[1]} == flash && $cword -eq 3 ]]; then
        local devices
        devices=$(_sg_fw_fetch_serial_devices)
        if [[ -z "$devices" ]]; then
            echo "No serial devices found" >&2
            return 0
        fi
        COMPREPLY=( $(compgen -W "$devices" -- "$cur") )
        return 0
    fi
}

complete -F _sg_fw sg-fw

