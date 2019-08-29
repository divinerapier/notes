#!/bin/bash

for i in {0..5000}; do
    openssl rand -out "${i}.txt" -base64 $(( 2**20 * 3 / 4 ));
    echo "${i}.txt" >> list.txt;
    if [ `expr $i % 100` -eq 0 ]
    then
        echo "processed ${i}";
    fi
done;
