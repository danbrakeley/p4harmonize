#!/bin/bash -e
required_minor_version=19

cd $(dirname "$0")

echo "Checking for go v1.$required_minor_version..."
if ! command -v go &> /dev/null
then
  echo "Could not find go. Make sure it is installed and in your path."
  echo "https://golang.org/dl/"
  exit 1
fi

go_version=$(go version | awk '{print $3}' | sed 's/^go//')
major_version=$(echo "$go_version" | cut -f1 -d.)
minor_version=$(echo "$go_version" | cut -f2 -d.)
if [[ "$major_version" -lt 1 || ( "$major_version" -eq 1 && "$minor_version" -lt "$required_minor_version" ) ]]; then
  echo "Need go version 1.$required_minor_version, you have $go_version. Please upgrade."
  echo "https://golang.org/dl/"
  exit 1
fi

echo "Checking for mage..."
if ! command -v mage &> /dev/null
then
  echo "Could not find mage. Installing..."
  ./reinstall-mage.sh
fi

echo "All dependencies are installed."