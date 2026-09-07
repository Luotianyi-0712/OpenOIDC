CREATE TABLE announcements (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    display_mode TEXT NOT NULL CHECK (display_mode IN ('banner', 'modal', 'both')),
    dismissible BOOLEAN NOT NULL DEFAULT TRUE,
    scrolling BOOLEAN NOT NULL DEFAULT FALSE,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    revision UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_announcements_published ON announcements (is_published, updated_at DESC);
