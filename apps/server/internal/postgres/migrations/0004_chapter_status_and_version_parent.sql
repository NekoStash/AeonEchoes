UPDATE chapters
SET status = 'drafting'
WHERE status = 'draft';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM chapters
        WHERE status NOT IN ('planned', 'drafting', 'reviewing', 'locked')
    ) THEN
        RAISE EXCEPTION 'chapters contain unsupported status values';
    END IF;
END $$;

ALTER TABLE chapters
    ALTER COLUMN status SET DEFAULT 'drafting';

ALTER TABLE chapters
    DROP CONSTRAINT IF EXISTS chapters_status_check;

ALTER TABLE chapters
    ADD CONSTRAINT chapters_status_check
    CHECK (status IN ('planned', 'drafting', 'reviewing', 'locked'));

DO $$
DECLARE
    bible_record RECORD;
    chapter_plan JSONB;
    normalized_plan JSONB;
BEGIN
    FOR bible_record IN
        SELECT id, source_seed #>> '{metadata,story_bible_chapter_plan}' AS raw_chapter_plan
        FROM story_bible_versions
        WHERE source_seed #> '{metadata,story_bible_chapter_plan}' IS NOT NULL
    LOOP
        BEGIN
            chapter_plan := bible_record.raw_chapter_plan::JSONB;
        EXCEPTION WHEN OTHERS THEN
            RAISE EXCEPTION 'story bible % contains invalid chapter plan JSON', bible_record.id;
        END;

        IF jsonb_typeof(chapter_plan) <> 'array' THEN
            RAISE EXCEPTION 'story bible % chapter plan is not an array', bible_record.id;
        END IF;

        IF EXISTS (
            SELECT 1
            FROM jsonb_array_elements(chapter_plan) AS item
            WHERE jsonb_typeof(item) <> 'object'
               OR NOT (item ? 'status')
               OR item ->> 'status' NOT IN ('draft', 'planned', 'drafting', 'reviewing', 'locked')
        ) THEN
            RAISE EXCEPTION 'story bible % chapter plan contains unsupported status values', bible_record.id;
        END IF;

        SELECT COALESCE(
            jsonb_agg(
                CASE
                    WHEN item ->> 'status' = 'draft'
                        THEN jsonb_set(item, '{status}', '"drafting"'::JSONB, false)
                    ELSE item
                END
                ORDER BY ordinal
            ),
            '[]'::JSONB
        )
        INTO normalized_plan
        FROM jsonb_array_elements(chapter_plan) WITH ORDINALITY AS plan_item(item, ordinal);

        UPDATE story_bible_versions
        SET source_seed = jsonb_set(
            source_seed,
            '{metadata,story_bible_chapter_plan}',
            to_jsonb(normalized_plan::TEXT),
            false
        )
        WHERE id = bible_record.id;
    END LOOP;
END $$;

ALTER TABLE chapter_versions
    ADD COLUMN parent_version_id TEXT;

UPDATE chapter_versions
SET parent_version_id = NULLIF(BTRIM(metadata ->> 'parent_version_id'), ''),
    metadata = metadata - 'parent_version_id'
WHERE metadata ? 'parent_version_id';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM chapter_versions child
        LEFT JOIN chapter_versions parent ON parent.id = child.parent_version_id
        WHERE child.parent_version_id IS NOT NULL
          AND (
              parent.id IS NULL
              OR parent.project_id <> child.project_id
              OR parent.chapter_id <> child.chapter_id
              OR parent.id = child.id
          )
    ) THEN
        RAISE EXCEPTION 'chapter_versions contain invalid parent references';
    END IF;

    IF EXISTS (
        WITH RECURSIVE version_chain AS (
            SELECT
                child.id AS origin_id,
                child.id AS current_id,
                child.parent_version_id AS next_id,
                ARRAY[child.id]::TEXT[] AS path,
                FALSE AS cycle
            FROM chapter_versions child
            WHERE child.parent_version_id IS NOT NULL

            UNION ALL

            SELECT
                chain.origin_id,
                parent.id,
                parent.parent_version_id,
                chain.path || parent.id,
                parent.id = ANY(chain.path)
            FROM version_chain chain
            JOIN chapter_versions parent ON parent.id = chain.next_id
            WHERE NOT chain.cycle
        )
        SELECT 1
        FROM version_chain
        WHERE cycle
    ) THEN
        RAISE EXCEPTION 'chapter_versions contain parent cycles';
    END IF;
END $$;

ALTER TABLE chapter_versions
    ADD CONSTRAINT uq_chapter_versions_parent_scope
    UNIQUE (id, project_id, chapter_id);

ALTER TABLE chapter_versions
    ADD CONSTRAINT fk_chapter_versions_parent_scope
    FOREIGN KEY (parent_version_id, project_id, chapter_id)
    REFERENCES chapter_versions(id, project_id, chapter_id)
    DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE chapter_versions
    ADD CONSTRAINT chapter_versions_parent_not_self
    CHECK (parent_version_id IS NULL OR parent_version_id <> id);

CREATE INDEX idx_chapter_versions_parent ON chapter_versions(parent_version_id);
