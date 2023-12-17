-- add test launcher_v2
BEGIN;
WITH entity
         AS ( INSERT INTO entities (id, entity_type, public) VALUES (gen_random_uuid(), 'launcher-v2', true) RETURNING id)
INSERT
INTO launcher_v2 (id, name)
SELECT id, 'Genesis'
FROM entity;
COMMIT;

-- add test files
BEGIN;
INSERT INTO files (id, entity_id, type, url, mime, size, uploaded_by, version, original_path, deployment_type, platform,
                   hash)
VALUES (gen_random_uuid(), '21023FA4-B853-4E85-BE78-7C290C557AF8'::uuid, 'test', 'https://veverse.com/test.jpg',
        'image/jpeg', 0, '1578BA66-3334-496E-8BB8-1A0696B42C68'::uuid, 0, 'test.jpg', 'Client', 'Win64', '');
INSERT INTO files (id, entity_id, type, url, mime, size, uploaded_by, version, original_path, deployment_type, platform,
                   hash)
VALUES (gen_random_uuid(), '21023FA4-B853-4E85-BE78-7C290C557AF8'::uuid, 'test', 'https://veverse.com/test.jpg',
        'image/jpeg', 0, '1578BA66-3334-496E-8BB8-1A0696B42C68'::uuid, 0, 'test.jpg', 'Client', '', '');
COMMIT;

-- add test owner accessible
INSERT INTO accessibles (entity_id, user_id, can_view, can_edit, can_delete, is_owner)
VALUES ('21023FA4-B853-4E85-BE78-7C290C557AF8'::uuid,
        '1578BA66-3334-496E-8BB8-1A0696B42C68'::uuid,
        true, true, true, true);

-- add test release_v2
BEGIN;
WITH entity
         AS (INSERT INTO entities (id, entity_type, public) VALUES (gen_random_uuid(), 'release-v2', true) RETURNING id)
INSERT
INTO release_v2 (id, entity_id, version, code_version, content_version, name, description)
SELECT id, '684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5'::uuid, '1.0.0', '1.0.0', '1.0.0', 'Genesis', 'Test Release 1.0.0'
FROM entity;
COMMIT;

-- test launcher_v2
SELECT l.id,
       l.name,
       e.created_at,
       e.updated_at,
       e.views,
       f.id,
       f.url,
       f.type,
       f.mime,
       f.original_path,
       f.size
FROM launcher_v2 l
         LEFT JOIN entities e ON l.id = e.id -- join entities to check public flag
         LEFT JOIN accessibles a ON a.entity_id = e.id AND a.user_id =
                                                           '1578BA66-3334-496E-8BB8-1A0696B42C68'::uuid -- join accessibles to check access
         LEFT JOIN files f on e.id = f.entity_id AND f.platform = 'Win64'::text -- join files
WHERE e.id = '684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5'::uuid
--   AND (e.public OR a.is_owner OR a.can_view)
--   AND f.platform = 'Win64'::text
ORDER BY e.created_at DESC;

-- test release_v2
SELECT r.id,
       r.name,
       r.description,
       r.version,
       r.entity_id,
       re.created_at,
       re.updated_at,
       re.views,
       ou.id,
       ou.name,
       f.id,
       f.url,
       f.type,
       f.mime,
       f.original_path,
       f.size
FROM release_v2 r
         LEFT JOIN entities le ON r.entity_id = le.id AND le.entity_type = 'launcher'
         LEFT JOIN accessibles oa ON le.id = oa.entity_id -- join accessibles to get owner
         LEFT JOIN users ou ON oa.user_id = ou.id AND oa.is_owner -- join users to get owner name
         LEFT JOIN entities re ON r.id = re.id -- join entities to check public flag
         LEFT JOIN files f on re.id = f.entity_id AND (f.platform = 'Win64'::text OR f.platform = '') -- join files
WHERE r.entity_id = '684D9ACB-C7B0-4FE6-BBAA-E2FA333B6DC5'::uuid
ORDER BY re.created_at DESC;