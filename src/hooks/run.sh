#!/bin/bash

if [ ! -d ../.git/hooks.local ]; then
  mv ../.git/hooks ../.git/hooks.local
  ln -s ../hooks ../.git/hooks
else
  echo "You already have hooks.local directory."
  echo "If it is the first time you run this script, then you should fix it so"
  echo "that hooks links to ../hooks and your own git hooks are in hooks.local."
fi
