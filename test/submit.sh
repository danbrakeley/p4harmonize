#!/bin/bash -e
cd $(dirname "$0")

source ./env.sh

PORT=$1
CL=$2

# commit the change left open by p4harmonize
p4 -p $PORT -u $DST_USER -c $DST_CLIENT submit -c $CL
