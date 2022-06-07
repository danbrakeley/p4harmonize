#!/bin/bash -e
cd $(dirname "$0")

echo "Checking for go..."
if ! command -v go &> /dev/null
then
  echo "Could not find go. Make sure it is installed and in your path."
  echo "https://golang.org/dl/"
  exit 1
fi

echo "Checking for mage..."
if ! command -v mage &> /dev/null
then
  echo "Could not find mage. Installing..."
  ./reinstall-mage.sh
fi

echo "All dependancies are installed."
