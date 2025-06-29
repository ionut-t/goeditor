SELECT * FROM users WHERE id = 1;
select * from users where id = 1;

SELECT id, name FROM users WHERE active = true ORDER BY created_at DESC LIMIT 10;
select id, name from users where active = true order by created_at desc limit 10;

SELECT id, name FROM users WHERE active = true ORDER BY created_at DESC LIMIT 10 OFFSET 5;
select id, name from users where active = true order by created_at desc limit 10 offset 5;


DELETE FROM users
WHERE created_at >= NOW() - INTERVAL '24 hours';

SELECT u.*
FROM public.users u
WHERE EXISTS (SELECT 1 FROM public.feeds f WHERE f.user_id = u.id AND f.created_at > '2024-01-01')
AND u.id IN (SELECT DISTINCT user_id FROM public.feeds WHERE created_at > '2024-01-01');

INSERT INTO users (id, created_at, updated_at, name, api_key)
SELECT
    gen_random_uuid(),
    NOW(),
    NOW(),
    'User ' || gs.i,
    encode(sha256((random())::text::bytea), 'hex'::text)
FROM generate_series(1, 1000) as gs(i);
