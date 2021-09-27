#!/bin/bash -e
cd $(dirname "$0")

SRC_PORT=1667
SRC_USER=super
SRC_CLIENT=super-UE4-Release-4.20

DST_PORT=1668
DST_USER=super
DST_CLIENT=super-test-engine-p4harmonize

SRC_P4="p4 -p $SRC_PORT -u $SRC_USER -c $SRC_CLIENT"
DST_P4="p4 -p $DST_PORT -u $DST_USER -c $DST_CLIENT"

# commit the change left open by p4harmonize

$DST_P4 submit -c 3

# grab the list of files from each server and compare names and types

SRCFILES=`$SRC_P4 -z tag files -e //$SRC_CLIENT/... |\
  grep -v "... time" |\
  grep -v "... rev" |\
  grep -v "... change" |\
  grep -v "... action" |\
  grep "... " |\
  sed -e 's/... //' |\
  sed -e 's|//UE4/Release-4.20/||' |\
  paste -d " "  - -`

DSTFILES=`$DST_P4 -z tag files -e //$DST_CLIENT/... |\
  grep -v "... time" |\
  grep -v "... rev" |\
  grep -v "... change" |\
  grep -v "... action" |\
  grep "... " |\
  sed -e 's/... //' |\
  sed -e 's|//test/engine/||' |\
  paste -d " "  - -`

if [[ "$SRCFILES" != "$DSTFILES" ]]; then
  echo "TEST FAILED: Source and Destination depots are not in sync"
  echo " --SOURCE--"
  echo "$SRCFILES"
  echo " --DESTINATION--"
  echo "$DSTFILES"
  exit 1
fi

echo "Source and Destination depots match"
