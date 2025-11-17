-- Migration Version: 20251116_001_add_coordinates_to_addresses.up.sql
-- Description: Add latitude and longitude columns to addresses table for warehouse distance calculation
-- Author: System Generated
-- Date: 2025-11-16

-- =====================================================
-- ADD COLUMNS: latitude và longitude
-- =====================================================
ALTER TABLE addresses 
ADD COLUMN latitude DECIMAL(9,6),
ADD COLUMN longitude DECIMAL(9,6);

-- =====================================================
-- CREATE INDEX for geographic queries
-- =====================================================
CREATE INDEX idx_addresses_location 
ON addresses(latitude, longitude) 
WHERE latitude IS NOT NULL AND longitude IS NOT NULL;

-- =====================================================
-- COMMENTS
-- =====================================================
COMMENT ON COLUMN addresses.latitude IS 'Latitude coordinate for warehouse distance calculation (optional)';
COMMENT ON COLUMN addresses.longitude IS 'Longitude coordinate for warehouse distance calculation (optional)';
