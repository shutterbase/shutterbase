#!/bin/bash

if ! command -v sops &> /dev/null
then
    echo "sops could not be found"
    exit 1
fi


all_secret_files=`find . -name "*secret.enc*"`

# first, add only stage-specific secrets
#echo Round 1: adding only stage-specific secrets
for file in $all_secret_files; do
    sops updatekeys -y $file
done