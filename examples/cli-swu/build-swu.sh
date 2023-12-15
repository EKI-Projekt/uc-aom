#!/bin/sh

SWU_NAME="example-uc-aom-cli"
FILES="sw-description callCliAddOnList.sh"

rm -f ${SWU_NAME}.swu
for i in $FILES;do
        echo $i;done | cpio -ov -H crc >  ${SWU_NAME}.swu
