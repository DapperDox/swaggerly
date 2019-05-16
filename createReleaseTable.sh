#!/usr/bin/env bash

VERSION=$1
DATE=`date "+%d-%m-%Y"`

## 1.0.0-beta (2016-10-25)

echo "**${VERSION} (${DATE})**"
echo ""

echo "| Filename | OS   | Arch | Size | Checksum |"
echo "| -------- | ---- | ---- | ---- | -------- |"

cd dist
for i in `ls`
do
    if [[ $i =~ release.*$ ]]
    then
        continue
    fi
    SUM=`shasum -pa256 $i | cut -f1 -d' '`
    SIZE=`du -h $i | cut -f1`
    TARG=`echo $i | cut -d'.' -f4`
    OS=`echo $TARG | cut -d'-' -f1`
    ARCH=`echo $TARG | cut -d'-' -f2`
    echo "[$i](https://github.com/frinka/dapperdox/releases/download/v$VERSION/$i) | ${OS} | ${ARCH} | ${SIZE} | ${SUM} |"
done
