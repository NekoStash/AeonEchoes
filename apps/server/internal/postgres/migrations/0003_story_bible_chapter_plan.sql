UPDATE story_bible_versions
SET source_seed = jsonb_set(
    source_seed,
    '{metadata}',
    CASE
        WHEN NOT ((source_seed -> 'metadata') ? 'story_bible_chapter_plan') THEN
            ((source_seed -> 'metadata') - 'story_bible_chapters')
            || jsonb_build_object(
                'story_bible_chapter_plan',
                (source_seed -> 'metadata' -> 'story_bible_chapters')
            )
        ELSE
            (source_seed -> 'metadata') - 'story_bible_chapters'
    END,
    true
)
WHERE jsonb_typeof(source_seed -> 'metadata') = 'object'
  AND (source_seed -> 'metadata') ? 'story_bible_chapters';
