#!/bin/bash -e
cd $(dirname "$0")

source ./env.sh

SRC_P4="p4 -p $1 -u $SRC_USER -c $SRC_CLIENT"
DST_P4="p4 -p $2 -u $DST_USER -c $DST_CLIENT"

# grab the list of files from each server and compare names and types

SRCFILES=`$SRC_P4 -z tag files -e //$SRC_CLIENT/... |\
  grep -v "... time" |\
  grep -v "... rev" |\
  grep -v "... change" |\
  grep -v "... action" |\
  grep "... " |\
  sed -e 's/... //' |\
  sed -e 's|//UE4/Release-4.20/||' |\
  paste -d " "  - - |\
  sort`

DSTFILES=`$DST_P4 -z tag files -e //$DST_CLIENT/... |\
  grep -v "... time" |\
  grep -v "... rev" |\
  grep -v "... change" |\
  grep -v "... action" |\
  grep "... " |\
  sed -e 's/... //' |\
  sed -e 's|//test/engine/||' |\
  paste -d " "  - - |\
  sort`

if [[ "$SRCFILES" != "$DSTFILES" ]]; then
  echo "TEST FAILED: Source and Destination depots are not in sync"
  echo " --SOURCE--"
  echo "$SRCFILES"
  echo " --DESTINATION--"
  echo "$DSTFILES"
  exit 1
fi

echo "Source and Destination depots match"
