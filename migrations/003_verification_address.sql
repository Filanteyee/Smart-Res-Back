-- Verification request теперь хранит адрес (entrance/floor/apartment),
-- который житель указывает в форме. При approve поля копируются в profiles
-- и используются для адресной FCM-доставки (см. internal/fcm/sender.go).

ALTER TABLE verification_requests ADD COLUMN IF NOT EXISTS entrance  INT;
ALTER TABLE verification_requests ADD COLUMN IF NOT EXISTS floor     INT;
ALTER TABLE verification_requests ADD COLUMN IF NOT EXISTS apartment VARCHAR(50);
