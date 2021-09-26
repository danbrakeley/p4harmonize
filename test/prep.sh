#!/bin/bash
cd $(dirname "$0")

if ! command -v cygpath > /dev/null; then
  echo "missing cygpath, are you on windows?"
  exit -1
fi

SRC_PORT=1667
SRC_USER=super
SRC_DEPOT=UE4
SRC_STREAM=Release-4.20
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

function add_file {
  echo "$5" > "$3"
  $1 add -c $2 -t $4 -f "$3"
}

function add_apple_file {
  AA_BASE=`basename "$3"`
  AA_DIR=`dirname "$3"`
  AA_FORK="$AA_DIR/%$AA_BASE"
  echo "AAUWBwACAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAJgAAABA=" | base64 -d > "$AA_FORK"
  printf "%s                " "$4" | cut -c -15 >> "$AA_FORK"
  echo "$5" > "$3"
  $1 add -c $2 -t apple -f "$3"
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
mkdir -p "$SRC_ROOT/Engine/Linux"
mkdir -p "$SRC_ROOT/Engine/Extras"
add_file "$SRC_P4" $CL "$SRC_ROOT/generate.cmd" "binary" "echo foo"
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/build.cs" "text" "// build stuff"
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/chair.uasset" "binary+l" "I'm a chair!"
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/door.uasset" "binary+l" "I'm a door!"
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/Linux/important.h" "text" "#include <frank.h>"
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/Linux/boring.h" "text" "#include <greg.h>"
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/Icon20@2x.png" "binary" "¯\\_(ツ)_/¯"
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/Icon30@2x.png" "binary" "¯\\_(ツ)_/¯"
add_file "$SRC_P4" $CL "$SRC_ROOT/Engine/Icon40@2x.png" "binary" "¯\\_(ツ)_/¯"
add_apple_file "$SRC_P4" $CL "$SRC_ROOT/Engine/Extras/Apple File.template" "resource fork" "this is just the data fork"
add_apple_file "$SRC_P4" $CL "$SRC_ROOT/Engine/Extras/Apple File Src.template" "source fork" "this is just the data fork"
add_apple_file "$SRC_P4" $CL "$SRC_ROOT/Engine/Extras/Borked.template" "resource fork" "this is just the data fork"

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
mkdir -p "$DST_ROOT/Engine/linux"
mkdir -p "$DST_ROOT/Engine/Extras"
add_file "$DST_P4" $CL "$DST_ROOT/generate.cmd" "text" "echo foo"
add_file "$DST_P4" $CL "$DST_ROOT/deprecated.txt" "utf8" "this file will be deleted very soon"
add_file "$DST_P4" $CL "$DST_ROOT/Engine/build.cs" "text" "// build stuff"
add_file "$DST_P4" $CL "$DST_ROOT/Engine/chair.uasset" "binary" "I'm a chair!"
add_file "$DST_P4" $CL "$DST_ROOT/Engine/rug.uasset" "binary" "I'm a rug!"
add_file "$DST_P4" $CL "$DST_ROOT/Engine/linux/important.h" "utf8" "#include <frank.h>"
add_file "$DST_P4" $CL "$DST_ROOT/Engine/linux/boring.h" "text" "#include <greg.h>"
add_file "$DST_P4" $CL "$DST_ROOT/Engine/Icon30@2x.png" "binary" "¯\\_(ツ)_/¯"
add_file "$DST_P4" $CL "$DST_ROOT/Engine/Icon40@2x.png" "binary" "image not found"
add_apple_file "$DST_P4" $CL "$DST_ROOT/Engine/Extras/Apple File.template" "i'm the resource fork" "this is just the data fork"
add_apple_file "$DST_P4" $CL "$DST_ROOT/Engine/Extras/Apple File Dst.template" "destination fork" "this is just the data fork"
add_file "$DST_P4" $CL "$DST_ROOT/Engine/Extras/Borked.template" "binary" "this is just the data fork"
add_file "$DST_P4" $CL "$DST_ROOT/Engine/Extras/%Borked.template" "binary" "this should never have been checked in"

$DST_P4 submit -c $CL
