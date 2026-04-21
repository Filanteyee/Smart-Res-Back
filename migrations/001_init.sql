CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS profiles (
    id                  UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    full_name           VARCHAR(255) DEFAULT '',
    email               VARCHAR(255) DEFAULT '',
    phone               VARCHAR(50)  DEFAULT '',
    iin                 VARCHAR(20)  DEFAULT '',
    person_type         VARCHAR(50)  DEFAULT '',
    city                VARCHAR(100) DEFAULT '',
    street              VARCHAR(255) DEFAULT '',
    property_type       VARCHAR(50)  DEFAULT '',
    property_number     VARCHAR(50)  DEFAULT '',
    full_address        TEXT         DEFAULT '',
    role                VARCHAR(50)  DEFAULT 'resident',
    verification_status VARCHAR(50)  DEFAULT 'not_submitted',
    created_at          TIMESTAMPTZ  DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS service_requests (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category    VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    status      VARCHAR(50)  DEFAULT 'new',
    created_at  TIMESTAMPTZ  DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS request_photos (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id UUID NOT NULL REFERENCES service_requests(id) ON DELETE CASCADE,
    file_path  VARCHAR(500) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS guest_access (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resident_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    guest_name   VARCHAR(255) NOT NULL,
    guest_phone  VARCHAR(50),
    car_number   VARCHAR(50),
    access_type  VARCHAR(20)  DEFAULT 'walk',
    access_code  VARCHAR(20)  NOT NULL,
    valid_from   TIMESTAMPTZ  NOT NULL,
    valid_until  TIMESTAMPTZ  NOT NULL,
    status       VARCHAR(20)  DEFAULT 'active',
    created_at   TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS barrier_logs (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID REFERENCES users(id),
    access_type VARCHAR(20),
    direction   VARCHAR(20),
    car_number  VARCHAR(50),
    notes       TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS verification_requests (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    requested_role VARCHAR(50)  NOT NULL,
    comment        TEXT         DEFAULT '',
    status         VARCHAR(50)  DEFAULT 'pending',
    reviewed_by    UUID         REFERENCES users(id),
    reviewed_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS verification_documents (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    verification_request_id UUID NOT NULL REFERENCES verification_requests(id) ON DELETE CASCADE,
    file_path               VARCHAR(500) NOT NULL,
    file_name               VARCHAR(255) NOT NULL,
    file_size               BIGINT       DEFAULT 0,
    created_at              TIMESTAMPTZ  DEFAULT NOW()
);
