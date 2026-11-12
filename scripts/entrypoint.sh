#!/usr/bin/env bash

set -euo pipefail
shopt -s nullglob

readonly DEFAULTS_DIR=/opt/ard
readonly DOCS_DIR="${ARD_DATAPATH:?ARD_DATAPATH is not set}/docs"
readonly SKILLS_DIR="${ARD_DATAPATH}/skills"

export ARD_SHELL="${ARD_SHELL:-/bin/bash}"
export ARD_SHELL_ATTRIBUTES="${ARD_SHELL_ATTRIBUTES:--c}"

##########################
# Entry point
#   Copy docs and run agent
# Arguments:
#   - None
##########################
main() {

    mkdir -p "$DOCS_DIR" "$SKILLS_DIR"

    # docs: always rewrite
    cp -rf "$DEFAULTS_DIR"/docs/. "$DOCS_DIR"/

    # skills: copy only unexisted
    for src in "$DEFAULTS_DIR"/skills/*; do
        dst="$SKILLS_DIR/${src##*/}"
        [[ -e "$dst" ]] && continue
        cp -r --no-preserve=mode "$src" "$dst"
    done

    exec "$DEFAULTS_DIR"/ard
}
main
