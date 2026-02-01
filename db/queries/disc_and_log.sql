SELECT
    s.disc_fingerprint_id,
    l.id AS make_mkv_log_id
FROM
    sessions s,
    json_each(l.args) arg
JOIN
    make_mkv_logs l ON s.id = l.session_id
WHERE
    -- Select only the first session (lowest ID) for each disc fingerprint
    s.id = (
        SELECT MIN(s2.id)
        FROM sessions s2
        WHERE s2.disc_fingerprint_id = s.disc_fingerprint_id
        AND s2.deleted_at IS NULL
    )
    -- Select only invocations that actually ran the makemkv info command
    AND arg.value = 'info'
    -- Ensure records are not soft-deleted
    AND s.deleted_at IS NULL
    AND l.deleted_at IS NULL;
