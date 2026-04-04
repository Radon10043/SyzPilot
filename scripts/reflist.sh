#!/bin/bash

for f in $(find $1 -name .tqueue); do basename $(dirname $f) | cut -d# -f1; done | sort