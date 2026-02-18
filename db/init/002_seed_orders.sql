INSERT INTO orders (id, customer_name, amount, status, created_at)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'Alice', 120, 'completed', NOW() - INTERVAL '30 minutes'),
    ('22222222-2222-2222-2222-222222222222', 'Bob', 340, 'created', NOW() - INTERVAL '20 minutes'),
    ('33333333-3333-3333-3333-333333333333', 'Charlie', 560, 'processing', NOW() - INTERVAL '10 minutes'),
    ('44444444-4444-4444-4444-444444444444', 'Diana', 220, 'completed', NOW() - INTERVAL '5 minutes'),
    ('55555555-5555-5555-5555-555555555555', 'Ethan', 90, 'created', NOW() - INTERVAL '1 minutes')
ON CONFLICT (id) DO NOTHING;