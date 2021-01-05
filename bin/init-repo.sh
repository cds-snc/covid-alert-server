#!/bin/bash

# Setup git hooks
git config core.hooksPath .githooks

# Install node dependencies (used for linting commits)
npm i
