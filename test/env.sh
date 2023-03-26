#!/bin/bash

SRC_USER=super
SRC_DEPOT=UE4
SRC_STREAM=Release-4.20
SRC_CLIENT=$SRC_USER-$SRC_DEPOT-$SRC_STREAM
SRC_ROOT=../local/p4/src

if command -v cygpath > /dev/null; then
  ## in cygwin-based bash (ie git bash) on windows
  SRC_ROOT_ABS=$(cygpath -a -w $SRC_ROOT | sed -e 's|\\|/|g')
elif command -v realpath > /dev/null; then
  SRC_ROOT_ABS=$(realpath -m $SRC_ROOT)
else
  echo "Unable to generate full path for $SRC_ROOT on this platform"
  exit 1
fi

DST_USER=super
DST_DEPOT=test
DST_STREAM=engine
DST_CLIENT=$DST_USER-$DST_DEPOT-$DST_STREAM-p4harmonize
DST_ROOT=../local/p4/dst

if command -v cygpath > /dev/null; then
  ## in cygwin-based bash (e.g. git bash) on windows
  DST_ROOT_ABS=$(cygpath -a -w $DST_ROOT | sed -e 's|\\|/|g')
elif command -v realpath > /dev/null; then
  DST_ROOT_ABS=$(realpath -m $DST_ROOT)
else
  echo "Unable to generate full path for $DST_ROOT on this platform"
  exit 1
fi


DST_INS_ROOT=../local/p4/dst_ins

if command -v cygpath > /dev/null; then
  ## in cygwin-based bash (e.g. git bash) on windows
  DST_INS_ROOT_ABS=$(cygpath -a -w $DST_INS_ROOT | sed -e 's|\\|/|g')
elif command -v realpath > /dev/null; then
  DST_INS_ROOT_ABS=$(realpath -m $DST_INS_ROOT)
else
  echo "Unable to generate full path for $DST_INS_ROOT on this platform"
  exit 1
fi
