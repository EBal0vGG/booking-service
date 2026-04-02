-- Dev seed: users (same UUIDs as dummy login), rooms, schedules.
-- Run after migrations: make seed

INSERT INTO users (id, email, role, created_at) VALUES
  ('11111111-1111-1111-1111-111111111111', 'admin@seed.local', 'admin', NOW()),
  ('22222222-2222-2222-2222-222222222222', 'user@seed.local', 'user', NOW())
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, role = EXCLUDED.role;

INSERT INTO rooms (id, name, description, capacity, created_at) VALUES
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Seed Room A', 'Conference A', 12, NOW()),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Seed Room B', NULL, 8, NOW()),
  ('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Seed Room C', 'Small', 4, NOW())
ON CONFLICT (id) DO NOTHING;

-- Room A: Mon–Wed 09:00–18:00; Room B: Mon only 10:00–16:00
INSERT INTO schedules (id, room_id, day_of_week, start_time, end_time, created_at) VALUES
  ('dddddddd-1111-1111-1111-111111111101', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 1, '09:00', '18:00', NOW()),
  ('dddddddd-1111-1111-1111-111111111102', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 2, '09:00', '18:00', NOW()),
  ('dddddddd-1111-1111-1111-111111111103', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 3, '09:00', '18:00', NOW()),
  ('dddddddd-2222-2222-2222-222222222201', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 1, '10:00', '16:00', NOW())
ON CONFLICT (id) DO NOTHING;
