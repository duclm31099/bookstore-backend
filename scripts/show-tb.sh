
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
docker exec -it bookstore_postgres psql -U $DB_USER -d $DB_NAME -c "\dt"

# \dt: Liệt kê tất cả tables (kiểm tra có users không).
# \l: Liệt kê tất cả databases.
# \du: Liệt kê users/roles.
# \q: Thoát psql.
# \h SELECT: Help về lệnh SQL.