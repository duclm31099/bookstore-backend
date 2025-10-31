#!/bin/bash

set -e  # Exit on error

echo "🗑️  Resetting database bookstore_dev..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Database credentials
DB_USER="bookstore"
DB_NAME="bookstore_dev"

# Step 1: Drop database
echo -e "${YELLOW}Step 1: Dropping database...${NC}"
docker exec -i bookstore_postgres psql -U $DB_USER -d postgres <<EOF
DROP DATABASE IF EXISTS $DB_NAME;
EOF

# Step 2: Create database
echo -e "${YELLOW}Step 2: Creating fresh database...${NC}"
docker exec -i bookstore_postgres psql -U $DB_USER -d postgres <<EOF
CREATE DATABASE $DB_NAME;
EOF

# Step 3: Run migrations
echo -e "${YELLOW}Step 3: Running migrations...${NC}"
make migrate-up

# Success
echo -e "${GREEN}✅ Database reset complete!${NC}"

# Show status
echo ""
echo -e "${YELLOW}Current status:${NC}"
make migrate-version

echo ""
echo -e "${YELLOW}Tables created:${NC}"
docker exec -it bookstore_postgres psql -U $DB_USER -d $DB_NAME -c "\dt"
