#!/bin/bash
cd $(dirname "$0")

if ! command -v cygpath > /dev/null; then
  echo "missing cygpath, are you on windows?"
  exit -1
fi

SRC_PORT=1667
SRC_USER=super
SRC_DEPOT=UE4
SRC_STREAM=Release-4.69
SRC_CLIENT=$SRC_USER-$SRC_DEPOT-$SRC_STREAM
SRC_ROOT=../local/p4/src
SRC_ROOT_WIN=$(cygpath -a -w $SRC_ROOT | sed -e 's|\\|/|g')

DST_PORT=1668
DST_USER=super
DST_DEPOT=test
DST_STREAM=engine
DST_CLIENT=$DST_USER-$DST_DEPOT-$DST_STREAM
DST_ROOT=../local/p4/dst
DST_ROOT_WIN=$(cygpath -a -w $DST_ROOT | sed -e 's|\\|/|g')

SRC_P4="p4 -p $SRC_PORT -u $SRC_USER"
DST_P4="p4 -p $DST_PORT -u $DST_USER"

if [[ "$1" == "clean" ]]; then
  $DST_P4 -c $DST_CLIENT obliterate -y //$DST_DEPOT/$DST_STREAM/...
  $DST_P4 client -d $DST_CLIENT
  $DST_P4 stream -d //$DST_DEPOT/$DST_STREAM
  $DST_P4 depot -d $DST_DEPOT
  $SRC_P4 -c $SRC_CLIENT obliterate -y //$SRC_DEPOT/$SRC_STREAM/...
  $SRC_P4 client -d $SRC_CLIENT
  $SRC_P4 stream -d //$SRC_DEPOT/$SRC_STREAM
  $SRC_P4 depot -d $SRC_DEPOT
  exit 0
fi

function add_file {
  # echo "add $3 with type $4 (using $1 with CL $2)"
  echo "foo" > "$3"
  $1 add -c $2 -t $4 "$3"
}

## add stuff to Source

$SRC_P4 --field "Type=stream" depot -o $SRC_DEPOT | $SRC_P4 depot -i
$SRC_P4 --field "Type=mainline" stream -o //$SRC_DEPOT/$SRC_STREAM | $SRC_P4 stream -i

$SRC_P4 \
  --field "Root=$SRC_ROOT_WIN" \
  --field "Stream=//$SRC_DEPOT/$SRC_STREAM" \
  --field "View=//$SRC_DEPOT/$SRC_STREAM/... //$SRC_CLIENT/..." \
  client -o $SRC_CLIENT | $SRC_P4 client -i


SRC_P4="$SRC_P4 -c $SRC_CLIENT"
CL=$($SRC_P4 --field "Description=test" --field "Files=" change -o | $SRC_P4 change -i | cut -d ' ' -f 2)

echo "Created CL $CL"

rm -rf "$SRC_ROOT"
mkdir -p "$SRC_ROOT/Engine"
add_file "$SRC_P4" $CL "$SRC_ROOT/generate.cmd" binary
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/build.cs" text
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/chair.uasset" binary+l
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/door.uasset" binary+l

$SRC_P4 submit -c $CL

## Add stuff to Destination

$DST_P4 --field "Type=stream" depot -o $DST_DEPOT | $DST_P4 depot -i
$DST_P4 --field "Type=mainline" stream -o //$DST_DEPOT/$DST_STREAM | $DST_P4 stream -i

$DST_P4 \
  --field "Root=$DST_ROOT_WIN" \
  --field "Stream=//$DST_DEPOT/$DST_STREAM" \
  --field "View=//$DST_DEPOT/$DST_STREAM/... //$DST_CLIENT/..." \
  client -o $DST_CLIENT | $DST_P4 client -i


DST_P4="$DST_P4 -c $DST_CLIENT"
CL=$($DST_P4 --field "Description=test" --field "Files=" change -o | $DST_P4 change -i | cut -d ' ' -f 2)

echo "Created CL $CL"

rm -rf "$DST_ROOT"
mkdir -p "$DST_ROOT/Engine"
add_file "$DST_P4" $CL "$DST_ROOT/generate.cmd" text
add_file "$DST_P4" $CL "$DST_ROOT/future.txt" utf8
add_file "$DST_P4" $CL "$DST_ROOT/Engine/build.cs" text
add_file "$DST_P4" $CL "$DST_ROOT/Engine/chair.uasset" binary
add_file "$DST_P4" $CL "$DST_ROOT/Engine/rug.uasset" binary

$DST_P4 submit -c $CL
