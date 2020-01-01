#!/bin/bash

# ================ 配置 ================
OWNER="lostvip-com"
REPO="lv_framework"
TOKEN=${TOKEN_GITHUB}
# ======================================

VERSION="$1"

if [ -z "$VERSION" ]; then
    echo "❌ 请提供版本号，如：./release.sh v1.0.0"
    exit 1
fi

# 创建并推送 tag
git tag -a "$VERSION" -m "Release $VERSION"
git push origin "$VERSION"

# 创建 GitHub Release
curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/$OWNER/$REPO/releases" \
  -d "{
        \"tag_name\": \"$VERSION\",
        \"name\": \"Release $VERSION\",
        \"body\": \"发布于 $(date '+%Y-%m-%d %H:%M:%S')\",
        \"draft\": false,
        \"prerelease\": false
     }"

echo
echo "✅ 已发布 Release: $VERSION"
echo "→ https://github.com/$OWNER/$REPO/releases/tag/$VERSION"
