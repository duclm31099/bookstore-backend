CREATE OR REPLACE FUNCTION release_stock(
    p_warehouse_id UUID,
    p_book_id UUID,
    p_quantity INT,
    p_user_id UUID DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_reserved INT;
BEGIN
    -- Fetch current reserved với FOR UPDATE để lock
    SELECT reserved
    INTO v_current_reserved
    FROM warehouse_inventory
    WHERE warehouse_id = p_warehouse_id
      AND book_id = p_book_id
    FOR UPDATE NOWAIT;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Inventory record not found';
    END IF;
    
    -- Validate có đủ reserved để release không
    IF v_current_reserved < p_quantity THEN
        RAISE EXCEPTION 'Cannot release % items, only % reserved', 
            p_quantity, v_current_reserved
            USING ERRCODE = 'BIZ02';
    END IF;
    
    -- Update: CHỈ set business fields, KHÔNG động version/updated_at
    UPDATE warehouse_inventory
    SET 
        reserved = GREATEST(reserved - p_quantity, 0),
        updated_by = p_user_id
        -- BỎ: version = v_current_version + 1
        -- BỎ: updated_at = NOW()
        -- Trigger sẽ tự động lo 2 field này
    WHERE warehouse_id = p_warehouse_id
      AND book_id = p_book_id;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION reserve_stock(
    p_warehouse_id UUID,
    p_book_id UUID,
    p_quantity INT,
    p_user_id UUID DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    v_available INT;
BEGIN
    SELECT (quantity - reserved)
    INTO v_available
    FROM warehouse_inventory
    WHERE warehouse_id = p_warehouse_id
      AND book_id = p_book_id
    FOR UPDATE NOWAIT;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Inventory record not found';
    END IF;
    
    IF v_available < p_quantity THEN
        RAISE EXCEPTION 'Insufficient stock: need %, have %', 
            p_quantity, v_available
            USING ERRCODE = 'BIZ01';
    END IF;
    
    UPDATE warehouse_inventory
    SET 
        reserved = reserved + p_quantity,
        updated_by = p_user_id
        -- BỎ version và updated_at
    WHERE warehouse_id = p_warehouse_id
      AND book_id = p_book_id;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;
