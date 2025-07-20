#!/bin/bash
set -euf -o pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

TARGET="unknown"
while [[ "$#" -gt 0 ]]; do
    case "$1" in
	--*)
	    shift
	    ;;
	invalid)
	    TARGET=drives.log
	    shift
	    ;;
	info)
	    TARGET=info.log
	    shift
	    ;;
	mkv)
	    TARGET=rip.log
	    shift
	    ;;
	*)
	    shift
	    ;;
    esac
done

cat "${SCRIPT_DIR}/${TARGET}"
