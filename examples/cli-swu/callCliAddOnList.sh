#!/bin/sh

echo "Calling uc-aom CLI via shellscript START"

/usr/bin/uc-aom list

if [ "$?" -eq 1 ]; then
    echo "Error: uc-aom was not found. Aborting process." >&2
    exit 1
fi

echo "Calling uc-aom CLI via shellscript END"
exit 0
