#!/bin/bash
cd $(dirname "$0")

if ! command -v cygpath > /dev/null; then
  echo "missing cygpath, are you on windows?"
  exit -1
fi

EPIC_PORT=1667
EPIC_USER=super
EPIC_DEPOT=UE4
EPIC_STREAM=Release-4.69
EPIC_CLIENT=$EPIC_USER-$EPIC_DEPOT-$EPIC_STREAM
EPIC_ROOT=../local/p4/src
EPIC_ROOT_WIN=$(cygpath -a -w $EPIC_ROOT | sed -e 's|\\|/|g')

LOCAL_PORT=1668
LOCAL_USER=super
LOCAL_DEPOT=test
LOCAL_STREAM=engine
LOCAL_CLIENT=$LOCAL_USER-$LOCAL_DEPOT-$LOCAL_STREAM
LOCAL_ROOT=../local/p4/dst
LOCAL_ROOT_WIN=$(cygpath -a -w $LOCAL_ROOT | sed -e 's|\\|/|g')

EPIC_P4="p4 -p $EPIC_PORT -u $EPIC_USER"
LOCAL_P4="p4 -p $LOCAL_PORT -u $LOCAL_USER"

if [[ "$1" == "clean" ]]; then
  $LOCAL_P4 -c $LOCAL_CLIENT obliterate -y //$LOCAL_DEPOT/$LOCAL_STREAM/...
  $LOCAL_P4 client -d $LOCAL_CLIENT
  $LOCAL_P4 stream -d //$LOCAL_DEPOT/$LOCAL_STREAM
  $LOCAL_P4 depot -d $LOCAL_DEPOT
  $EPIC_P4 -c $EPIC_CLIENT obliterate -y //$EPIC_DEPOT/$EPIC_STREAM/...
  $EPIC_P4 client -d $EPIC_CLIENT
  $EPIC_P4 stream -d //$EPIC_DEPOT/$EPIC_STREAM
  $EPIC_P4 depot -d $EPIC_DEPOT
  exit 0
fi

function add_file {
  # echo "add $3 with type $4 (using $1 with CL $2)"
  echo "foo" > "$3"
  $1 add -c $2 -t $4 "$3"
}

## add stuff to Epic

$EPIC_P4 --field "Type=stream" depot -o $EPIC_DEPOT | $EPIC_P4 depot -i
$EPIC_P4 --field "Type=mainline" stream -o //$EPIC_DEPOT/$EPIC_STREAM | $EPIC_P4 stream -i

$EPIC_P4 \
  --field "Root=$EPIC_ROOT_WIN" \
  --field "Stream=//$EPIC_DEPOT/$EPIC_STREAM" \
  --field "View=//$EPIC_DEPOT/$EPIC_STREAM/... //$EPIC_CLIENT/..." \
  client -o $EPIC_CLIENT | $EPIC_P4 client -i


EPIC_P4="$EPIC_P4 -c $EPIC_CLIENT"
CL=$($EPIC_P4 --field "Description=test" --field "Files=" change -o | $EPIC_P4 change -i | cut -d ' ' -f 2)

echo "Created CL $CL"

rm -rf "$EPIC_ROOT"
mkdir -p "$EPIC_ROOT/Engine"
add_file "$EPIC_P4" $CL "$EPIC_ROOT/generate.cmd" binary
add_file "$EPIC_P4" $CL "$EPIC_ROOT/Engine/build.cs" text
add_file "$EPIC_P4" $CL "$EPIC_ROOT/Engine/chair.uasset" binary+l
add_file "$EPIC_P4" $CL "$EPIC_ROOT/Engine/door.uasset" binary+l

$EPIC_P4 submit -c $CL

## Add stuff to Local

$LOCAL_P4 --field "Type=stream" depot -o $LOCAL_DEPOT | $LOCAL_P4 depot -i
$LOCAL_P4 --field "Type=mainline" stream -o //$LOCAL_DEPOT/$LOCAL_STREAM | $LOCAL_P4 stream -i

$LOCAL_P4 \
  --field "Root=$LOCAL_ROOT_WIN" \
  --field "Stream=//$LOCAL_DEPOT/$LOCAL_STREAM" \
  --field "View=//$LOCAL_DEPOT/$LOCAL_STREAM/... //$LOCAL_CLIENT/..." \
  client -o $LOCAL_CLIENT | $LOCAL_P4 client -i


LOCAL_P4="$LOCAL_P4 -c $LOCAL_CLIENT"
CL=$($LOCAL_P4 --field "Description=test" --field "Files=" change -o | $LOCAL_P4 change -i | cut -d ' ' -f 2)

echo "Created CL $CL"

rm -rf "$LOCAL_ROOT"
mkdir -p "$LOCAL_ROOT/Engine"
add_file "$LOCAL_P4" $CL "$LOCAL_ROOT/generate.cmd" text
add_file "$LOCAL_P4" $CL "$LOCAL_ROOT/future.txt" utf8
add_file "$LOCAL_P4" $CL "$LOCAL_ROOT/Engine/build.cs" text
add_file "$LOCAL_P4" $CL "$LOCAL_ROOT/Engine/chair.uasset" binary
add_file "$LOCAL_P4" $CL "$LOCAL_ROOT/Engine/rug.uasset" binary

$LOCAL_P4 submit -c $CL
