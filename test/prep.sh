#!/bin/bash
cd $(dirname "$0")

source ./env.sh

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

## $1 is the server port
## $2 is the server root (relative)
## $3 is the server root (absolute, windows)
function add_src_files {
  local PORT=$1
  local ROOT=$2
  local ROOT_ABS=$3
  local SRC_P4="p4 -p $PORT -u $SRC_USER"

  $SRC_P4 --field "Type=stream" depot -o $SRC_DEPOT | $SRC_P4 depot -i
  $SRC_P4 --field "Type=mainline" stream -o //$SRC_DEPOT/$SRC_STREAM | $SRC_P4 stream -i

  $SRC_P4 \
    --field "Root=$ROOT_ABS" \
    --field "Stream=//$SRC_DEPOT/$SRC_STREAM" \
    --field "View=//$SRC_DEPOT/$SRC_STREAM/... //$SRC_CLIENT/..." \
    client -o $SRC_CLIENT | $SRC_P4 client -i

  SRC_P4="$SRC_P4 -c $SRC_CLIENT"
  local CL=$($SRC_P4 --field "Description=test" --field "Files=" change -o | $SRC_P4 change -i | cut -d ' ' -f 2)

  echo "Created CL $CL"

  rm -rf "$ROOT"
  mkdir -p "$ROOT/Engine/Linux"
  mkdir -p "$ROOT/Engine/Extras"
  add_file "$SRC_P4" $CL "$ROOT/generate.cmd" "binary" "echo foo"
  add_file "$SRC_P4" $CL "$ROOT/Engine/build.cs" "text" "// build stuff"
  add_file "$SRC_P4" $CL "$ROOT/Engine/chair.uasset" "binary+l" "I'm a chair!"
  add_file "$SRC_P4" $CL "$ROOT/Engine/door.uasset" "binary+l" "I'm a door!"
  add_file "$SRC_P4" $CL "$ROOT/Engine/Linux/important.h" "text" "#include <frank.h>"
  add_file "$SRC_P4" $CL "$ROOT/Engine/Linux/boring.h" "text" "#include <greg.h>"
  add_file "$SRC_P4" $CL "$ROOT/Engine/Icon20@2x.png" "binary" "¯\\_(ツ)_/¯"
  add_file "$SRC_P4" $CL "$ROOT/Engine/Icon30@2x.png" "binary" "¯\\_(ツ)_/¯"
  add_file "$SRC_P4" $CL "$ROOT/Engine/Icon40@2x.png" "binary" "¯\\_(ツ)_/¯"
  add_apple_file "$SRC_P4" $CL "$ROOT/Engine/Extras/Apple File.template" "resource fork" "this is just the data fork"
  add_apple_file "$SRC_P4" $CL "$ROOT/Engine/Extras/Apple File Src.template" "source fork" "this is just the data fork"
  add_apple_file "$SRC_P4" $CL "$ROOT/Engine/Extras/Borked.template" "resource fork" "this is just the data fork"

  $SRC_P4 submit -c $CL
}

## $1 is the server port
## $2 is the server root (relative)
## $3 is the server root (absolute, windows)
function add_dst_files {
  local PORT=$1
  local ROOT=$2
  local ROOT_ABS=$3
  local DST_P4="p4 -p $PORT -u $DST_USER"
  local DST_CLIENT_ADD=$DST_USER-$DST_DEPOT-$DST_STREAM

  $DST_P4 --field "Type=stream" depot -o $DST_DEPOT | $DST_P4 depot -i
  $DST_P4 --field "Type=mainline" stream -o //$DST_DEPOT/$DST_STREAM | $DST_P4 stream -i

  $DST_P4 \
    --field "Root=$ROOT_ABS" \
    --field "Stream=//$DST_DEPOT/$DST_STREAM" \
    --field "View=//$DST_DEPOT/$DST_STREAM/... //$DST_CLIENT_ADD/..." \
    client -o $DST_CLIENT_ADD | $DST_P4 client -i


  DST_P4="$DST_P4 -c $DST_CLIENT_ADD"
  local CL=$($DST_P4 --field "Description=test" --field "Files=" change -o | $DST_P4 change -i | cut -d ' ' -f 2)

  echo "Created CL $CL"

  rm -rf "$ROOT"
  mkdir -p "$ROOT/Engine/linux"
  mkdir -p "$ROOT/Engine/Extras"
  add_file "$DST_P4" $CL "$ROOT/generate.cmd" "text" "echo foo"
  add_file "$DST_P4" $CL "$ROOT/deprecated.txt" "utf8" "this file will be deleted very soon"
  add_file "$DST_P4" $CL "$ROOT/Engine/build.cs" "text" "// build stuff"
  add_file "$DST_P4" $CL "$ROOT/Engine/chair.uasset" "binary" "I'm a chair!"
  add_file "$DST_P4" $CL "$ROOT/Engine/rug.uasset" "binary" "I'm a rug!"
  add_file "$DST_P4" $CL "$ROOT/Engine/linux/important.h" "utf8" "#include <frank.h>"
  add_file "$DST_P4" $CL "$ROOT/Engine/linux/boring.h" "text" "#include <greg.h>"
  add_file "$DST_P4" $CL "$ROOT/Engine/Icon30@2x.png" "binary" "¯\\_(ツ)_/¯"
  add_file "$DST_P4" $CL "$ROOT/Engine/Icon40@2x.png" "binary" "image not found"
  add_apple_file "$DST_P4" $CL "$ROOT/Engine/Extras/Apple File.template" "i'm the resource fork" "this is just the data fork"
  add_apple_file "$DST_P4" $CL "$ROOT/Engine/Extras/Apple File Dst.template" "destination fork" "this is just the data fork"
  add_file "$DST_P4" $CL "$ROOT/Engine/Extras/Borked.template" "binary" "this is just the data fork"
  add_file "$DST_P4" $CL "$ROOT/Engine/Extras/%Borked.template" "binary" "this should never have been checked in"

  $DST_P4 submit -c $CL
}

## do the work

add_src_files 1667 "$SRC_ROOT" "$SRC_ROOT_ABS"
add_dst_files 1668 "$DST_ROOT" "$DST_ROOT_ABS"
add_dst_files 1669 "$DST_INS_ROOT" "$DST_INS_ROOT_ABS"
