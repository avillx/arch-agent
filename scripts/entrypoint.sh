#!/bin/sh


##########################
# Entry point
#   Copy docs and run agent
# Arguments:
#   - None
##########################
main() {

    # copy docs
    mkdir -p "$DATA_PATH"/docs
    cp /arch/docs/* "$DATA_PATH"/docs

    # run agent
    ./arch/arch-agent
}
main
