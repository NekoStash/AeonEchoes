CREATE TABLE IF NOT EXISTS skill_sources (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('inline_text', 'directory')),
    path TEXT NOT NULL DEFAULT '',
    inline_text TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_skill_sources_project ON skill_sources(project_id);
CREATE INDEX IF NOT EXISTS idx_skill_sources_enabled ON skill_sources(enabled);
CREATE INDEX IF NOT EXISTS idx_skill_sources_type ON skill_sources(type);

CREATE TABLE IF NOT EXISTS skills (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    source_id TEXT NOT NULL REFERENCES skill_sources(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_skills_source ON skills(source_id);
CREATE INDEX IF NOT EXISTS idx_skills_project ON skills(project_id);
CREATE INDEX IF NOT EXISTS idx_skills_enabled ON skills(enabled);

CREATE TABLE IF NOT EXISTS mcp_server_configs (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    transport TEXT NOT NULL CHECK (transport IN ('stdio', 'streamable_http', 'sse')),
    status TEXT NOT NULL DEFAULT 'unknown' CHECK (status IN ('online', 'offline', 'disabled', 'failed', 'unknown')),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    command TEXT NOT NULL DEFAULT '',
    args JSONB NOT NULL DEFAULT '[]'::jsonb,
    url TEXT NOT NULL DEFAULT '',
    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    secret_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    env JSONB NOT NULL DEFAULT '{}'::jsonb,
    secret_env JSONB NOT NULL DEFAULT '{}'::jsonb,
    timeout_sec INTEGER NOT NULL DEFAULT 0 CHECK (timeout_sec >= 0),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mcp_server_configs_project ON mcp_server_configs(project_id);
CREATE INDEX IF NOT EXISTS idx_mcp_server_configs_enabled_status ON mcp_server_configs(enabled, status);

CREATE TABLE IF NOT EXISTS agent_configs (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT '',
    model_id TEXT REFERENCES model_configs(id) ON DELETE SET NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    system_prompt TEXT NOT NULL DEFAULT '',
    skill_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    tool_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    mcp_server_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    memory_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    runtime_options JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_configs_project ON agent_configs(project_id);
CREATE INDEX IF NOT EXISTS idx_agent_configs_enabled ON agent_configs(enabled);

CREATE TABLE IF NOT EXISTS agent_runs (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agent_configs(id) ON DELETE CASCADE,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    output JSONB NOT NULL DEFAULT '{}'::jsonb,
    error TEXT NOT NULL DEFAULT '',
    tool_invocation_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_agent ON agent_runs(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_runs_project_status ON agent_runs(project_id, status);

CREATE TABLE IF NOT EXISTS tool_definitions (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL CHECK (kind IN ('builtin', 'mcp', 'skill')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'unavailable')),
    mcp_server_id TEXT REFERENCES mcp_server_configs(id) ON DELETE SET NULL,
    source_id TEXT REFERENCES skill_sources(id) ON DELETE SET NULL,
    skill_id TEXT REFERENCES skills(id) ON DELETE SET NULL,
    input_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, name, kind)
);

CREATE INDEX IF NOT EXISTS idx_tool_definitions_project ON tool_definitions(project_id);
CREATE INDEX IF NOT EXISTS idx_tool_definitions_kind_status ON tool_definitions(kind, status);
CREATE INDEX IF NOT EXISTS idx_tool_definitions_mcp_server ON tool_definitions(mcp_server_id);
CREATE INDEX IF NOT EXISTS idx_tool_definitions_source ON tool_definitions(source_id);

CREATE TABLE IF NOT EXISTS tool_invocations (
    id TEXT PRIMARY KEY,
    agent_run_id TEXT REFERENCES agent_runs(id) ON DELETE SET NULL,
    agent_id TEXT REFERENCES agent_configs(id) ON DELETE SET NULL,
    project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    tool_id TEXT REFERENCES tool_definitions(id) ON DELETE SET NULL,
    tool_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'succeeded', 'failed')),
    arguments JSONB NOT NULL DEFAULT '{}'::jsonb,
    result JSONB NOT NULL DEFAULT '{}'::jsonb,
    error TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tool_invocations_agent_run ON tool_invocations(agent_run_id);
CREATE INDEX IF NOT EXISTS idx_tool_invocations_agent ON tool_invocations(agent_id);
CREATE INDEX IF NOT EXISTS idx_tool_invocations_project ON tool_invocations(project_id);
CREATE INDEX IF NOT EXISTS idx_tool_invocations_tool ON tool_invocations(tool_id);
CREATE INDEX IF NOT EXISTS idx_tool_invocations_status ON tool_invocations(status);
