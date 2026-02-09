-- Create buying_lists table
CREATE TABLE IF NOT EXISTS buying_lists (
    id BIGSERIAL PRIMARY KEY,
    family_id BIGINT REFERENCES families(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT 'Shopping List',
    created_by_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_buying_lists_chat_id ON buying_lists(chat_id);
CREATE INDEX IF NOT EXISTS idx_buying_lists_family_id ON buying_lists(family_id);

-- Create buying_items table
CREATE TABLE IF NOT EXISTS buying_items (
    id BIGSERIAL PRIMARY KEY,
    buying_list_id BIGINT NOT NULL REFERENCES buying_lists(id) ON DELETE CASCADE,
    name VARCHAR(500) NOT NULL,
    quantity VARCHAR(100),
    bought BOOLEAN DEFAULT false,
    bought_by_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    added_by_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_buying_items_list_id ON buying_items(buying_list_id);
CREATE INDEX IF NOT EXISTS idx_buying_items_bought ON buying_items(buying_list_id, bought);
