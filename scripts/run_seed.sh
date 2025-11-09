#!/bin/bash

# Database connection info
DB_HOST="localhost"
DB_PORT="5439"
DB_NAME="bookstore_dev"
DB_USER="bookstore"
DB_PASSWORD="secret"

# Export password to avoid prompt
export PGPASSWORD=$DB_PASSWORD

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo "Starting seed data insertion..."
echo "========================================="

# Array of seed files in correct order
SEED_FILES=(
    "001_users_seed.sql"
    "002_book.sql"
    "003_inventory.sql"
    "004_promotion.sql"
    "005_order.sql"
    "006_payment.sql"
    "007_refund.sql"
    "008_review.sql"
)

# Loop through and execute each file
for seed_file in "${SEED_FILES[@]}"
do
    echo -e "\n${YELLOW}Running: $seed_file${NC}"
    
    if psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f "seeds/$seed_file"; then
        echo -e "${GREEN}✓ Success: $seed_file${NC}"
    else
        echo -e "${RED}✗ Failed: $seed_file${NC}"
        echo -e "${RED}Stopping seed process due to error${NC}"
        exit 1
    fi
done

# Unset password
unset PGPASSWORD

echo -e "\n========================================="
echo -e "${GREEN}All seed data inserted successfully!${NC}"
echo "========================================="
