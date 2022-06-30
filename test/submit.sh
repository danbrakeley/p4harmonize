#!/bin/bash -e
cd $(dirname "$0")

source ./env.sh

# commit the change left open by p4harmonize
p4 -p $DST_PORT -u $DST_USER -c $DST_CLIENT submit -c 3
