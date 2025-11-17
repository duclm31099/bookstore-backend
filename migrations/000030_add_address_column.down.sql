-- Migration Version: 20251116_001_add_coordinates_to_addresses.down.sql
-- Description: Rollback coordinates columns from addresses table
-- Author: System Generated
-- Date: 2025-11-16

-- Remove index
DROP INDEX IF EXISTS idx_addresses_location;

-- Remove columns
ALTER TABLE addresses 
DROP COLUMN IF EXISTS latitude,
DROP COLUMN IF EXISTS longitude;
