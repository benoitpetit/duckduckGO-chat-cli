#!/bin/bash

# Script to fix the version issue: delete v1.1.10 and prepare for v1.2.0
# This script fixes the auto-increment issue that created 1.1.10 instead of 1.2.0

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔧 Fixing version issue: removing v1.1.10 and preparing for v1.2.0${NC}"
echo ""

# Check if we're in the right repository
if ! git remote get-url origin | grep -q 'duckduckGO-chat-cli'; then
    echo -e "${RED}❌ Not in the correct repository${NC}"
    exit 1
fi

# Check current branch
CURRENT_BRANCH=$(git branch --show-current)
echo -e "${BLUE}Current branch: $CURRENT_BRANCH${NC}"

# Delete the incorrect v1.1.10 tag locally if it exists
if git tag -l | grep -q "^v1\.1\.10$"; then
    echo -e "${YELLOW}🗑️ Deleting local tag v1.1.10...${NC}"
    git tag -d v1.1.10
    echo -e "${GREEN}✅ Local tag v1.1.10 deleted${NC}"
else
    echo -e "${GREEN}✅ Local tag v1.1.10 doesn't exist${NC}"
fi

# Delete the incorrect v1.1.10 tag from remote if it exists
if git ls-remote --tags origin | grep -q "refs/tags/v1\.1\.10"; then
    echo -e "${YELLOW}🗑️ Deleting remote tag v1.1.10...${NC}"
    git push --delete origin v1.1.10 || echo -e "${YELLOW}⚠️ Remote tag v1.1.10 might have been deleted already${NC}"
    echo -e "${GREEN}✅ Remote tag v1.1.10 deleted${NC}"
else
    echo -e "${GREEN}✅ Remote tag v1.1.10 doesn't exist${NC}"
fi

# Delete the incorrect v1.1.10 release on GitHub if it exists
echo -e "${YELLOW}📦 Checking for GitHub release v1.1.10...${NC}"
echo -e "${BLUE}ℹ️ You may need to manually delete the GitHub release v1.1.10 at:${NC}"
echo -e "${BLUE}   https://github.com/benoitpetit/duckduckGO-chat-cli/releases/tag/v1.1.10${NC}"

# Check if v1.2.0 tag already exists
if git tag -l | grep -q "^v1\.2\.0$"; then
    echo -e "${YELLOW}⚠️ Local tag v1.2.0 already exists${NC}"
    echo -n "Delete it and recreate? (y/n): "
    read -r CONFIRM
    if [[ $CONFIRM == "y" || $CONFIRM == "Y" ]]; then
        git tag -d v1.2.0
        echo -e "${GREEN}✅ Local tag v1.2.0 deleted${NC}"
    fi
fi

if git ls-remote --tags origin | grep -q "refs/tags/v1\.2\.0"; then
    echo -e "${YELLOW}⚠️ Remote tag v1.2.0 already exists${NC}"
    echo -n "Delete it from remote and recreate? (y/n): "
    read -r CONFIRM
    if [[ $CONFIRM == "y" || $CONFIRM == "Y" ]]; then
        git push --delete origin v1.2.0 || echo -e "${YELLOW}⚠️ Could not delete remote tag${NC}"
        echo -e "${GREEN}✅ Remote tag v1.2.0 deleted${NC}"
    fi
fi

echo ""
echo -e "${GREEN}🎉 Version cleanup complete!${NC}"
echo ""
echo -e "${BLUE}📋 Next steps:${NC}"
echo "1. The workflow has been updated to detect version 1.2.0 from PR titles"
echo "2. Create a PR with title containing 'v1.2.0' to the 'prod' branch"
echo "3. The workflow will now use 1.2.0 instead of auto-incrementing"
echo ""
echo -e "${YELLOW}💡 PR title should be: '🚀 Release v1.2.0 - Intelligent Features & Enhanced Stability'${NC}"
echo ""
echo -e "${BLUE}🔗 Manual cleanup if needed:${NC}"
echo "- Delete GitHub release: https://github.com/benoitpetit/duckduckGO-chat-cli/releases/tag/v1.1.10"
echo "- The updated workflow will now prioritize version detection from PR titles" 