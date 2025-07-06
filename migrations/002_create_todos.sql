-- Create todos table
CREATE TABLE IF NOT EXISTS todos (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    priority VARCHAR(10) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high')),
    deadline TIMESTAMP WITH TIME ZONE,
    created_by_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_to_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    chat_id BIGINT NOT NULL,
    message_id BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_todos_chat_id ON todos(chat_id);
CREATE INDEX IF NOT EXISTS idx_todos_assigned_to ON todos(assigned_to_id) WHERE assigned_to_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status);
CREATE INDEX IF NOT EXISTS idx_todos_priority ON todos(priority);
CREATE INDEX IF NOT EXISTS idx_todos_deadline ON todos(deadline) WHERE deadline IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_todos_created_by ON todos(created_by_id);

-- Create composite index for common queries
CREATE INDEX IF NOT EXISTS idx_todos_chat_status ON todos(chat_id, status);
CREATE INDEX IF NOT EXISTS idx_todos_assigned_status ON todos(assigned_to_id, status) WHERE assigned_to_id IS NOT NULL;