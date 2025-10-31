CREATE TABLE IF NOT EXISTS order_status_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    from_status TEXT,
    to_status TEXT NOT NULL,
    
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    notes TEXT,
    
    changed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_order_status_history_order ON order_status_history(order_id, changed_at DESC);
CREATE INDEX idx_order_status_history_changed_by ON order_status_history(changed_by);

-- Auto-track status changes
CREATE OR REPLACE FUNCTION track_order_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'UPDATE' AND OLD.status != NEW.status) THEN
        INSERT INTO order_status_history (order_id, from_status, to_status, notes)
        VALUES (NEW.id, OLD.status, NEW.status, 'Auto-tracked');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_track_order_status
    AFTER UPDATE OF status ON orders
    FOR EACH ROW
    EXECUTE FUNCTION track_order_status_change();
