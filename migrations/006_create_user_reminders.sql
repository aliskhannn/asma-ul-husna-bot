-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_reminders
(
    user_id        bigint PRIMARY KEY,
    is_enabled     boolean             DEFAULT TRUE,
    interval_hours smallint            DEFAULT 1 CHECK (interval_hours IN (1, 2, 3, 4)),
    start_time     varchar(8) NOT NULL DEFAULT '08:00:00',
    end_time       varchar(8) NOT NULL DEFAULT '20:00:00',
    last_sent_at   timestamptz         DEFAULT NULL,
    next_send_at   timestamptz         DEFAULT NULL,
    created_at     timestamptz         DEFAULT NOW(),
    updated_at     timestamptz         DEFAULT NOW(),

    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE OR REPLACE FUNCTION calculate_next_send_at(
    p_timezone text,
    p_start_time text,
    p_interval_hours integer,
    p_now timestamptz
) RETURNS timestamptz as $$
DECLARE
    user_local_time timestamptz;
    start_hour integer;
    current_hour integer;
    next_hour integer;
    next_time timestamptz;
BEGIN
    -- Convert to user timezone
    user_local_time := p_now AT TIME ZONE p_timezone;

    -- Extract start hour
    start_hour := EXTRACT(HOUR FROM p_start_time::TIME);
    current_hour := EXTRACT(HOUR FROM user_local_time);

    -- Find next valid hour
    IF current_hour < start_hour THEN
        next_hour := start_hour;
    ELSE
        -- Calculate next interval-aligned hour
        next_hour := start_hour +
                     ((current_hour - start_hour) / p_interval_hours + 1) * p_interval_hours;
    END IF;

    -- Construct next send time
    next_time := date_trunc('hour', user_local_time) + (next_hour || ' hours')::INTERVAL;

    -- If next_hour is tomorrow, add a day
    IF next_hour >= 24 THEN
        next_time := next_time + INTERVAL '1 day';
    END IF;

    RETURN next_time AT TIME ZONE p_timezone;
END;
$$ LANGUAGE plpgsql;

CREATE INDEX idx_reminders_enabled ON user_reminders (is_enabled, last_sent_at)
    WHERE is_enabled = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_reminders_enabled;

DROP TABLE IF EXISTS user_reminders;
-- +goose StatementEnd
