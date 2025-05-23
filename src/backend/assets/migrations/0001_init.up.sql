CREATE TABLE IF NOT EXISTS configs (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    user_id SERIAL PRIMARY KEY,
    user_name TEXT NOT NULL UNIQUE,
    hashed_password TEXT NOT NULL UNIQUE,
    hashed_cookie_value TEXT UNIQUE,
    cookie_expiration_date TEXT,
    is_admin BOOLEAN NOT NULL,
    -- is ignored in open core, just used in premium edition
    did_accept_eula BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS apps (
    app_id SERIAL PRIMARY KEY,
    maintainer TEXT NOT NULL,
    app_name TEXT NOT NULL,
    version_name TEXT NOT NULL,
    version_creation_timestamp TEXT NOT NULL,
    version_content BYTEA NOT NULL,
    should_be_running BOOLEAN NOT NULL,
    UNIQUE (maintainer, app_name)
);
