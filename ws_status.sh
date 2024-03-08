#!/usr/bin/env bash
# Generates the data used by the stamping feature in bazel.

echo STABLE_GIT_COMMIT "$(git rev-parse HEAD)"
