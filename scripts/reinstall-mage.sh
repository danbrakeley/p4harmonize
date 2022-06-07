#!/bin/bash -e
cd $(dirname "$0")

TMPFOLDER=$(mktemp -d) || exit 1
trap 'rm -rf "$TMPFOLDER"' EXIT

if command -v mage &> /dev/null
then
  echo "--- MAGE VERSION BEFORE"
  mage -version
  echo "---"
fi

cd $TMPFOLDER
git clone https://github.com/magefile/mage
cd mage
go run bootstrap.go

echo "--- MAGE VERSION"
mage -version
echo "---"
