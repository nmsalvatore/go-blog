#!/bin/bash
set -e

# Resolve the directory where this script resides
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Assume the project root is one level up
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Switch to project root context for building and syncing
cd "$PROJECT_ROOT"

# Configuration
# 1. Try to load from .env.deploy (check deploy/ folder first, then root)
if [ -f "${SCRIPT_DIR}/.env.deploy" ]; then
    echo -e "📜 Loading configuration from ${SCRIPT_DIR}/.env.deploy"
    export $(grep -v '^#' "${SCRIPT_DIR}/.env.deploy" | xargs)
elif [ -f .env.deploy ]; then
    echo -e "📜 Loading configuration from .env.deploy"
    export $(grep -v '^#' .env.deploy | xargs)
fi

# 2. Set variables (Env vars take precedence over defaults)
USER="${DEPLOY_USER:-user}"
HOST="${DEPLOY_HOST:-example.com}"
DIR="${DEPLOY_DIR:-/opt/blog}"
BINARY_NAME="blog"
SERVICE_NAME="blog"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting deployment to ${USER}@${HOST}:${DIR}...${NC}"

# 1. Run Tests locally
echo -e "${GREEN}🧪 Running tests...${NC}"
go test ./...
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Tests failed. Aborting deployment.${NC}"
    exit 1
fi

# 2. Build for Linux
echo -e "${GREEN}🏗️  Building binary for Linux...${NC}"
GOOS=linux GOARCH=amd64 go build -o ${BINARY_NAME} .
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Build failed. Aborting deployment.${NC}"
    exit 1
fi

# 3. Create remote directory if it doesn't exist
echo -e "${GREEN}📂 Ensuring remote directory exists...${NC}"
ssh ${USER}@${HOST} "mkdir -p ${DIR}"

# 4. Sync files
# We exclude:
# - .env (contains production secrets)
# - blog.db (production database)
# - .git/ (git history)
# - tests (not needed in prod)
# - deploy/ (deployment scripts not needed in prod)
echo -e "${GREEN}📡 Syncing files...${NC}"
rsync -avz --progress \
    --exclude '.env' \
    --exclude 'blog.db' \
    --exclude '.git' \
    --exclude '*_test.go' \
    --exclude 'deploy/' \
    --exclude 'README.md' \
    ./ ${USER}@${HOST}:${DIR}

# 5. Restart Service
echo -e "${GREEN}🔄 Restarting systemd service...${NC}"
ssh ${USER}@${HOST} "sudo systemctl restart ${SERVICE_NAME}"

echo -e "${GREEN}✅ Deployment complete!${NC}"
