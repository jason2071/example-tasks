CREATE TABLE
    IF NOT EXISTS example.tasks (
        "id" BIGSERIAL PRIMARY KEY,
        "title" text NOT NULL,
        "status" varchar(20) DEFAULT 'pending',
        "created_at" timestamptz DEFAULT CURRENT_TIMESTAMP,
        "updated_at" timestamptz DEFAULT CURRENT_TIMESTAMP,
        "deleted_at" timestamptz NULL,
        CONSTRAINT "check_campaign_status" CHECK (status IN ('pending', 'doing', 'done'))
    );

-- Drop existing unique key constraint on title if exists
ALTER TABLE example.tasks
DROP CONSTRAINT IF EXISTS uq_example_tasks_ref;

-- Add deleted_at column for soft delete on existing databases
ALTER TABLE example.tasks
ADD COLUMN IF NOT EXISTS "deleted_at" timestamptz NULL;

-- Replace hard unique title with active-record unique index for soft delete
DROP INDEX IF EXISTS uq_example_tasks_ref;
CREATE UNIQUE INDEX IF NOT EXISTS uq_example_tasks_title_active ON example.tasks (title)
WHERE deleted_at IS NULL;

-- Add priority column
ALTER TABLE example.tasks
ADD COLUMN IF NOT EXISTS "priority" int NULL;

-- Create index on priority column
CREATE INDEX IF NOT EXISTS idx_example_tasks_priority ON example.tasks (priority);

-- Add check constraint for priority range (1 to 5)
ALTER TABLE example.tasks ADD CONSTRAINT "check_priority_range" CHECK (
    priority IS NULL
    OR (
        priority >= 1
        AND priority <= 5
    )
);