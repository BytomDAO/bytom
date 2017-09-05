#!/bin/bash

if [ "$1" = "bytom0" ];
then
    ./bytom node --home ./test/.bytom0
elif [ "$1" = "bytom1" ];
then
    ./bytom node --home ./test/.bytom1
else
    echo "please cin -----./test.sh bytom0[bytom1]------ ."
fi
