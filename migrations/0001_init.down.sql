DROP INDEX IF EXISTS uq_bookings_active_slot;
DROP INDEX IF EXISTS idx_bookings_slot_id;
DROP INDEX IF EXISTS idx_bookings_user_active;
DROP INDEX IF EXISTS idx_bookings_user_id;
DROP TABLE IF EXISTS bookings;

DROP INDEX IF EXISTS idx_slots_room_id_start_time;
DROP TABLE IF EXISTS slots;

DROP INDEX IF EXISTS idx_schedules_room_id;
DROP TABLE IF EXISTS schedules;

DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS users;
