#!/bin/bash
# Daily OpenClaw Intelligence Gathering v2
# Sources: GitHub API + Tavily AI Search (replaces blocked Reddit/Twitter/etc)
# Output: markdown report for Lyra to process

OUTPUT_DIR="/home/claw/.openclaw/workspace/memory/openclaw-intel"
TAVILY_SCRIPT="/home/claw/.openclaw/workspace/skills/tavily-search/scripts/search.mjs"
mkdir -p "$OUTPUT_DIR"
DATE=$(date +%Y-%m-%d)
OUTPUT="$OUTPUT_DIR/$DATE.md"
YESTERDAY=$(date -d '1 day ago' +%Y-%m-%d 2>/dev/null || date -v-1d +%Y-%m-%d 2>/dev/null)

# Load API key
source ~/.bashrc 2>/dev/null
export TAVILY_API_KEY="${TAVILY_API_KEY:-}"

echo "# OpenClaw Intelligence Report - $DATE" > "$OUTPUT"
echo "" >> "$OUTPUT"

# =============================================
# SECTION 1: GitHub API Sources (free, no quota)
# =============================================

# 1. awesome-openclaw-usecases (commits + new usecases)
echo "## 1. Use Cases (awesome-openclaw-usecases)" >> "$OUTPUT"
echo "### Recent Commits" >> "$OUTPUT"
curl -s "https://api.github.com/repos/hesamsheikh/awesome-openclaw-usecases/commits?since=${YESTERDAY}T00:00:00Z&per_page=10" 2>/dev/null | python3 -c "
import json,sys
try:
    data=json.load(sys.stdin)
    if isinstance(data, list) and len(data)>0:
        for c in data:
            msg=c.get('commit',{}).get('message','').split('\n')[0]
            date=c.get('commit',{}).get('author',{}).get('date','')
            print(f'- [{date}] {msg}')
    else:
        print('No new commits')
except:
    print('Error fetching')
" >> "$OUTPUT" 2>/dev/null
echo "" >> "$OUTPUT"

# Fetch full usecase list and highlight new/updated ones
echo "### All Use Cases (for reference)" >> "$OUTPUT"
curl -s "https://raw.githubusercontent.com/hesamsheikh/awesome-openclaw-usecases/main/README.md" 2>/dev/null | python3 -c "
import sys,re
content=sys.stdin.read()
# Extract usecase links
links=re.findall(r'\[([^\]]+)\]\(.*?usecases/([^\)]+)\.md\)', content)
categories=re.findall(r'## ([^\n]+)', content)
for name,slug in links:
    print(f'- {name} ({slug})')
if not links:
    print('Could not parse usecases')
" >> "$OUTPUT" 2>/dev/null
echo "" >> "$OUTPUT"

# 2. awesome-openclaw-skills
echo "## 2. Skills Repo Updates" >> "$OUTPUT"
curl -s "https://api.github.com/repos/VoltAgent/awesome-openclaw-skills/commits?since=${YESTERDAY}T00:00:00Z&per_page=10" 2>/dev/null | python3 -c "
import json,sys
try:
    data=json.load(sys.stdin)
    if isinstance(data, list) and len(data)>0:
        for c in data:
            msg=c.get('commit',{}).get('message','').split('\n')[0]
            date=c.get('commit',{}).get('author',{}).get('date','')
            print(f'- [{date}] {msg}')
    else:
        print('No new commits')
except:
    print('Error fetching')
" >> "$OUTPUT" 2>/dev/null
echo "" >> "$OUTPUT"

# 3. OpenClaw releases
echo "## 3. OpenClaw Core Releases" >> "$OUTPUT"
curl -s "https://api.github.com/repos/openclaw/openclaw/releases?per_page=3" 2>/dev/null | python3 -c "
import json,sys
try:
    data=json.load(sys.stdin)
    if isinstance(data, list):
        for r in data[:3]:
            tag=r.get('tag_name','')
            date=r.get('published_at','')[:10]
            body=(r.get('body','') or '')[:200]
            print(f'### {tag} ({date})')
            print(body)
            print()
except:
    print('Error fetching')
" >> "$OUTPUT" 2>/dev/null
echo "" >> "$OUTPUT"

# 4. Claude Code changelog
echo "## 4. Claude Code Changelog" >> "$OUTPUT"
curl -s "https://code.claude.com/docs/en/changelog.md" 2>/dev/null | head -50 >> "$OUTPUT"
echo "" >> "$OUTPUT"

# 5. GitHub trending openclaw repos
echo "## 5. Trending OpenClaw Repos" >> "$OUTPUT"
curl -s "https://api.github.com/search/repositories?q=openclaw+pushed:>${YESTERDAY}&sort=stars&order=desc&per_page=10" 2>/dev/null | python3 -c "
import json,sys
try:
    data=json.load(sys.stdin)
    for r in data.get('items',[])[:10]:
        name=r['full_name']
        stars=r['stargazers_count']
        desc=(r.get('description','') or '')[:100]
        url=r.get('html_url','')
        print(f'- ⭐{stars} **{name}** — {desc}')
except:
    print('Error fetching')
" >> "$OUTPUT" 2>/dev/null
echo "" >> "$OUTPUT"

# =============================================
# SECTION 2: Tavily AI Search (quota: ~15/day)
# =============================================

if [ -n "$TAVILY_API_KEY" ]; then
    echo "## 6. AI Agent Industry Trends (via Tavily)" >> "$OUTPUT"
    echo "" >> "$OUTPUT"

    # 6a. OpenClaw ecosystem news
    echo "### 6a. OpenClaw Ecosystem" >> "$OUTPUT"
    node "$TAVILY_SCRIPT" "OpenClaw AI agent new features updates 2026" --topic news --days 7 -n 5 >> "$OUTPUT" 2>/dev/null
    echo "" >> "$OUTPUT"

    # 6b. Claude Code / Anthropic developer tools
    echo "### 6b. Claude Code & Anthropic Dev Tools" >> "$OUTPUT"
    node "$TAVILY_SCRIPT" "Claude Code CLI agent coding updates 2026" --topic news --days 7 -n 5 >> "$OUTPUT" 2>/dev/null
    echo "" >> "$OUTPUT"

    # 6c. AI agent frameworks & multi-agent systems
    echo "### 6c. AI Agent Frameworks & Multi-Agent" >> "$OUTPUT"
    node "$TAVILY_SCRIPT" "AI agent framework multi-agent orchestration latest 2026" --topic news --days 7 -n 5 >> "$OUTPUT" 2>/dev/null
    echo "" >> "$OUTPUT"

    # 6d. AI coding assistants competition
    echo "### 6d. AI Coding Tools Landscape" >> "$OUTPUT"
    node "$TAVILY_SCRIPT" "AI coding assistant Cursor Copilot Windsurf comparison 2026" --topic news --days 7 -n 5 >> "$OUTPUT" 2>/dev/null
    echo "" >> "$OUTPUT"

    # 6e. Smart glasses / AR industry (nimo competitive intel)
    echo "### 6e. Smart Glasses & AR Industry" >> "$OUTPUT"
    node "$TAVILY_SCRIPT" "smart glasses AR wearable AI 2026 latest" --topic news --days 7 -n 5 >> "$OUTPUT" 2>/dev/null
    echo "" >> "$OUTPUT"

    # 6f. AI infrastructure & best practices
    echo "### 6f. AI Ops & Best Practices" >> "$OUTPUT"
    node "$TAVILY_SCRIPT" "AI agent deployment production best practices DevOps 2026" -n 5 >> "$OUTPUT" 2>/dev/null
    echo "" >> "$OUTPUT"

    # 6g. Chinese AI industry (国内AI动态)
    echo "### 6g. 国内AI动态" >> "$OUTPUT"
    node "$TAVILY_SCRIPT" "中国 AI 智能体 agent 最新进展 2026" --topic news --days 7 -n 5 >> "$OUTPUT" 2>/dev/null
    echo "" >> "$OUTPUT"

    echo "_(Tavily: ~7 queries used, ~2000/month quota)_" >> "$OUTPUT"
else
    echo "## 6. Tavily Search" >> "$OUTPUT"
    echo "⚠️ TAVILY_API_KEY not set, skipping AI search" >> "$OUTPUT"
fi

echo "" >> "$OUTPUT"
echo "---" >> "$OUTPUT"
echo "Report generated at $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$OUTPUT"

echo "Done: $OUTPUT"
