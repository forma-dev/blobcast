#!/bin/bash

# Check if tag argument is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <tag>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

TAG=$1

# check if tag already exists
if git tag -l | grep -q $TAG; then
    echo "Tag $TAG already exists"
    exit 1
fi

# Create tag and push to github
git tag -a $TAG -m "Release $TAG"
git push origin $TAG
