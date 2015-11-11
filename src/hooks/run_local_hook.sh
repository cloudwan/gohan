#!/bin/bash

HOOK_NAME=$(ps -o comm= $PPID)
HOOK_PATH=".git/hooks.local/$HOOK_NAME"
if [ -f $HOOK_PATH ]; then
  ./$HOOK_PATH $@
fi
