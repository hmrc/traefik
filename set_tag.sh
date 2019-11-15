#!/bin/bash

CURRENT_LATEST_TAG=$(git tag --sort=-version:refname | grep "release\/*" | head -1)

VERSION=$(echo $CURRENT_LATEST_TAG | awk -F'/' '{print $2}')

echo "Last version: $VERSION"

MAJOR=$(echo $VERSION | awk -F'.' '{print $1}')
MINOR=$(echo $VERSION | awk -F'.' '{print $2}')

MINOR=$[ $MINOR + 1 ]

NEW_VERSION="$MAJOR.$MINOR.0"

echo "New Version: $NEW_VERSION"

git tag release/${NEW_VERSION}
git push --tags
