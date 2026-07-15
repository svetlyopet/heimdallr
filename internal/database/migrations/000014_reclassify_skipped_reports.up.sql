UPDATE reports
SET status = 'failed',
    updated_at = NOW()
WHERE status = 'skipped';
