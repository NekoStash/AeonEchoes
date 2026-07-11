-- Remove narrative builtins that are no longer part of agent.NarrativeToolSpecs.
-- Runtime seed also deletes any other obsolete builtins and scrubs agent tool_ids.
DELETE FROM tool_definitions
WHERE kind = 'builtin'
  AND (
    id IN ('builtin:chapter.ensure', 'builtin:chapter.create')
    OR name IN ('chapter.ensure', 'chapter.create')
  );

-- Drop agent_configs.tool_ids entries that reference the removed builtins.
UPDATE agent_configs
SET tool_ids = COALESCE((
    SELECT jsonb_agg(value)
    FROM jsonb_array_elements(tool_ids) AS value
    WHERE value #>> '{}' NOT IN (
        'builtin:chapter.ensure',
        'builtin:chapter.create',
        'builtin.chapter.ensure',
        'builtin.chapter.create',
        'chapter.ensure',
        'chapter.create'
    )
    AND btrim(value #>> '{}') <> ''
), '[]'::jsonb),
    updated_at = now()
WHERE tool_ids @> '["builtin:chapter.ensure"]'::jsonb
   OR tool_ids @> '["builtin:chapter.create"]'::jsonb
   OR tool_ids @> '["builtin.chapter.ensure"]'::jsonb
   OR tool_ids @> '["builtin.chapter.create"]'::jsonb
   OR tool_ids @> '["chapter.ensure"]'::jsonb
   OR tool_ids @> '["chapter.create"]'::jsonb
   OR tool_ids @> '[""]'::jsonb;
