#!/usr/bin/env bash

# loop for 5 times:
#   echo "Iteration NR in $1" to either stdout or stderr (random)
#   sleep for random time between 0 < t <= 1 secs

for i in $(seq 1 5)
do
  x=$(( ( $RANDOM % 2 )  + 1 ))
  y="0.$(( $RANDOM % 9 ))"
  (>&$x echo "Instance $1 iteration loop $i")
  sh -c "sleep $y"
  if [ $x -eq 1 ]; then x=2; else x=1; fi
done
