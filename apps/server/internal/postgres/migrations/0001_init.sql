CREATE TABLE IF NOT EXISTS provider_configs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('openai-responses', 'openai', 'anthropic', 'gemini')),
    base_url TEXT NOT NULL DEFAULT '',
    api_key_ciphertext TEXT NOT NULL DEFAULT '',
    api_key_env TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    trace_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    trace_retention_days INTEGER NOT NULL DEFAULT 14 CHECK (trace_retention_days >= 0),
    default_request_timeout_sec INTEGER NOT NULL DEFAULT 60 CHECK (default_request_timeout_sec > 0),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_model_refresh_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_provider_configs_type_enabled ON provider_configs(type, enabled);

CREATE TABLE IF NOT EXISTS model_configs (
    id TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL REFERENCES provider_configs(id) ON DELETE CASCADE,
    provider_type TEXT NOT NULL CHECK (provider_type IN ('openai-responses', 'openai', 'anthropic', 'gemini')),
    name TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL CHECK (kind IN ('text', 'embedding')),
    context_window INTEGER NOT NULL DEFAULT 0 CHECK (context_window >= 0),
    max_output_tokens INTEGER NOT NULL DEFAULT 0 CHECK (max_output_tokens >= 0),
    dimension INTEGER NOT NULL DEFAULT 0 CHECK (dimension >= 0),
    supports_tools BOOLEAN NOT NULL DEFAULT FALSE,
    supports_streaming BOOLEAN NOT NULL DEFAULT FALSE,
    default_for_kind BOOLEAN NOT NULL DEFAULT FALSE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cost_input_per_mtok NUMERIC(12, 6) NOT NULL DEFAULT 0,
    cost_output_per_mtok NUMERIC(12, 6) NOT NULL DEFAULT 0,
    routing_weight INTEGER NOT NULL DEFAULT 100,
    allowed_agent_roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider_id, name, kind)
);

CREATE INDEX IF NOT EXISTS idx_model_configs_kind_enabled ON model_configs(kind, enabled);
CREATE INDEX IF NOT EXISTS idx_model_configs_provider ON model_configs(provider_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_model_configs_default_text ON model_configs(kind) WHERE default_for_kind = TRUE AND enabled = TRUE;

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    seed JSONB NOT NULL DEFAULT '{}'::jsonb,
    active_story_bible_id TEXT,
    default_worldline_id TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
CREATE INDEX IF NOT EXISTS idx_projects_slug ON projects(slug);

CREATE TABLE IF NOT EXISTS story_bible_versions (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version INTEGER NOT NULL CHECK (version > 0),
    title TEXT NOT NULL,
    logline TEXT NOT NULL DEFAULT '',
    synopsis TEXT NOT NULL DEFAULT '',
    genre TEXT NOT NULL DEFAULT '',
    tone TEXT NOT NULL DEFAULT '',
    audience TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT 'zh-CN',
    themes JSONB NOT NULL DEFAULT '[]'::jsonb,
    rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    worldline_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    entity_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    plot_thread_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_seed JSONB NOT NULL DEFAULT '{}'::jsonb,
    genesis_workflow_id TEXT NOT NULL DEFAULT '',
    approved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, version)
);

CREATE INDEX IF NOT EXISTS idx_story_bible_versions_project ON story_bible_versions(project_id, version DESC);

ALTER TABLE projects
    ADD CONSTRAINT fk_projects_active_story_bible
    FOREIGN KEY (active_story_bible_id)
    REFERENCES story_bible_versions(id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE IF NOT EXISTS worldlines (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    canonical BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_worldlines_project ON worldlines(project_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_worldlines_canonical ON worldlines(project_id) WHERE canonical = TRUE;

CREATE TABLE IF NOT EXISTS narrative_entities (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    worldline_id TEXT REFERENCES worldlines(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    aliases JSONB NOT NULL DEFAULT '[]'::jsonb,
    summary TEXT NOT NULL DEFAULT '',
    traits JSONB NOT NULL DEFAULT '{}'::jsonb,
    importance INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_narrative_entities_project ON narrative_entities(project_id);
CREATE INDEX IF NOT EXISTS idx_narrative_entities_worldline ON narrative_entities(worldline_id);
CREATE INDEX IF NOT EXISTS idx_narrative_entities_type ON narrative_entities(type);

CREATE TABLE IF NOT EXISTS narrative_facts (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    worldline_id TEXT REFERENCES worldlines(id) ON DELETE SET NULL,
    entity_id TEXT REFERENCES narrative_entities(id) ON DELETE SET NULL,
    chapter_id TEXT NOT NULL DEFAULT '',
    chapter_version_id TEXT NOT NULL DEFAULT '',
    claim TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    confidence NUMERIC(5, 4) NOT NULL DEFAULT 1.0 CHECK (confidence >= 0 AND confidence <= 1),
    status TEXT NOT NULL DEFAULT 'active',
    embedding_ref TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_narrative_facts_project ON narrative_facts(project_id);
CREATE INDEX IF NOT EXISTS idx_narrative_facts_entity ON narrative_facts(entity_id);
CREATE INDEX IF NOT EXISTS idx_narrative_facts_status ON narrative_facts(status);

CREATE TABLE IF NOT EXISTS graph_edges (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    worldline_id TEXT REFERENCES worldlines(id) ON DELETE SET NULL,
    source_entity_id TEXT NOT NULL REFERENCES narrative_entities(id) ON DELETE CASCADE,
    target_entity_id TEXT NOT NULL REFERENCES narrative_entities(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    label TEXT NOT NULL DEFAULT '',
    weight NUMERIC(8, 4) NOT NULL DEFAULT 1.0,
    evidence_fact_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_graph_edges_project ON graph_edges(project_id);
CREATE INDEX IF NOT EXISTS idx_graph_edges_source ON graph_edges(source_entity_id);
CREATE INDEX IF NOT EXISTS idx_graph_edges_target ON graph_edges(target_entity_id);

CREATE TABLE IF NOT EXISTS plot_threads (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    worldline_id TEXT REFERENCES worldlines(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open',
    priority INTEGER NOT NULL DEFAULT 0,
    related_entity_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    opened_chapter_id TEXT NOT NULL DEFAULT '',
    closed_chapter_id TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_plot_threads_project_status ON plot_threads(project_id, status);

CREATE TABLE IF NOT EXISTS chapters (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number INTEGER NOT NULL CHECK (number > 0),
    title TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, number)
);

CREATE INDEX IF NOT EXISTS idx_chapters_project ON chapters(project_id, number);

CREATE TABLE IF NOT EXISTS chapter_versions (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    chapter_id TEXT NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    version INTEGER NOT NULL CHECK (version > 0),
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    author_role TEXT NOT NULL DEFAULT 'writer',
    source_workflow_id TEXT NOT NULL DEFAULT '',
    index_status TEXT NOT NULL DEFAULT 'pending',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(chapter_id, version)
);

CREATE INDEX IF NOT EXISTS idx_chapter_versions_project ON chapter_versions(project_id);
CREATE INDEX IF NOT EXISTS idx_chapter_versions_index_status ON chapter_versions(index_status);

CREATE TABLE IF NOT EXISTS index_jobs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    chapter_id TEXT REFERENCES chapters(id) ON DELETE CASCADE,
    chapter_version_id TEXT REFERENCES chapter_versions(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    scheduled_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_index_jobs_project_status ON index_jobs(project_id, status);
CREATE INDEX IF NOT EXISTS idx_index_jobs_kind_status ON index_jobs(kind, status);

CREATE TABLE IF NOT EXISTS ai_workflows (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    model_id TEXT REFERENCES model_configs(id) ON DELETE SET NULL,
    context_pack_id TEXT NOT NULL DEFAULT '',
    steps JSONB NOT NULL DEFAULT '[]'::jsonb,
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    output JSONB NOT NULL DEFAULT '{}'::jsonb,
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_workflows_project_kind ON ai_workflows(project_id, kind);
CREATE INDEX IF NOT EXISTS idx_ai_workflows_status ON ai_workflows(status);

CREATE TABLE IF NOT EXISTS ai_runs (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES ai_workflows(id) ON DELETE CASCADE,
    provider_id TEXT NOT NULL REFERENCES provider_configs(id) ON DELETE RESTRICT,
    model_id TEXT NOT NULL REFERENCES model_configs(id) ON DELETE RESTRICT,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    latency_millis INTEGER NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    trace_object_ref TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_runs_workflow ON ai_runs(workflow_id);
CREATE INDEX IF NOT EXISTS idx_ai_runs_provider_model ON ai_runs(provider_id, model_id);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_settings_updated_at ON settings(updated_at);
