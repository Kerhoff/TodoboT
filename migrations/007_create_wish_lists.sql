-- Create wish_lists table
CREATE TABLE IF NOT EXISTS wish_lists (
    id BIGSERIAL PRIMARY KEY,
    family_id BIGINT REFERENCES families(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT 'My Wishes',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wish_lists_family_id ON wish_lists(family_id);
CREATE INDEX IF NOT EXISTS idx_wish_lists_user_id ON wish_lists(user_id);

-- Create wish_items table
CREATE TABLE IF NOT EXISTS wish_items (
    id BIGSERIAL PRIMARY KEY,
    wish_list_id BIGINT NOT NULL REFERENCES wish_lists(id) ON DELETE CASCADE,
    name VARCHAR(500) NOT NULL,
    url TEXT,
    price VARCHAR(100),
    notes TEXT,
    reserved BOOLEAN DEFAULT false,
    reserved_by_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wish_items_list_id ON wish_items(wish_list_id);
CREATE INDEX IF NOT EXISTS idx_wish_items_reserved ON wish_items(wish_list_id, reserved);
