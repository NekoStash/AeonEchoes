'use strict';

const http = require('node:http');
const { URL } = require('node:url');
const crypto = require('node:crypto');

const host = process.env.FAKE_PROVIDER_HOST || '127.0.0.1';
const port = Number(process.env.FAKE_PROVIDER_PORT || 8787);

const openAiModels = [
  { id: 'fake-gpt-4.1', object: 'model', created: 1_700_000_000, owned_by: 'fake-provider' },
  { id: 'fake-gpt-4.1-mini', object: 'model', created: 1_700_000_000, owned_by: 'fake-provider' },
  { id: 'fake-embedding-3-small', object: 'model', created: 1_700_000_000, owned_by: 'fake-provider' }
];

const anthropicModels = [
  { id: 'fake-claude-3-5-sonnet', type: 'model', display_name: 'Fake Claude 3.5 Sonnet', created_at: '2024-01-01T00:00:00Z' },
  { id: 'fake-claude-3-haiku', type: 'model', display_name: 'Fake Claude 3 Haiku', created_at: '2024-01-01T00:00:00Z' }
];

const geminiModels = [
  {
    name: 'models/fake-gemini-1.5-pro',
    displayName: 'Fake Gemini 1.5 Pro',
    supportedGenerationMethods: ['generateContent', 'streamGenerateContent']
  },
  {
    name: 'models/fake-embedding-001',
    displayName: 'Fake Gemini Embedding',
    supportedGenerationMethods: ['embedContent', 'batchEmbedContents']
  }
];

function jsonResponse(res, statusCode, body, headers = {}) {
  const payload = JSON.stringify(body);
  res.writeHead(statusCode, {
    'content-type': 'application/json; charset=utf-8',
    'content-length': Buffer.byteLength(payload),
    ...headers
  });
  res.end(payload);
}

function textResponse(res, statusCode, body, headers = {}) {
  res.writeHead(statusCode, {
    'content-type': 'text/plain; charset=utf-8',
    'content-length': Buffer.byteLength(body),
    ...headers
  });
  res.end(body);
}

function sseResponse(res, chunks) {
  res.writeHead(200, {
    'content-type': 'text/event-stream; charset=utf-8',
    'cache-control': 'no-cache, no-transform',
    connection: 'keep-alive'
  });
  for (const chunk of chunks) {
    res.write(`data: ${JSON.stringify(chunk)}\n\n`);
  }
  res.write('data: [DONE]\n\n');
  res.end();
}

function notFound(res, path) {
  jsonResponse(res, 404, {
    error: {
      type: 'not_found',
      message: `fake-provider has no route for ${path}`
    }
  });
}

function methodNotAllowed(res, method) {
  jsonResponse(res, 405, {
    error: {
      type: 'method_not_allowed',
      message: `method ${method} is not supported for this route`
    }
  });
}

function readBody(req) {
  return new Promise((resolve, reject) => {
    let body = '';
    req.setEncoding('utf8');
    req.on('data', chunk => {
      body += chunk;
      if (body.length > 2 * 1024 * 1024) {
        reject(new Error('request body exceeds 2 MiB limit'));
        req.destroy();
      }
    });
    req.on('end', () => {
      if (!body.trim()) {
        resolve({});
        return;
      }
      try {
        resolve(JSON.parse(body));
      } catch (error) {
        reject(new Error(`invalid JSON request body: ${error.message}`));
      }
    });
    req.on('error', reject);
  });
}

function stableId(prefix, value) {
  const hash = crypto.createHash('sha256').update(JSON.stringify(value)).digest('hex').slice(0, 24);
  return `${prefix}_${hash}`;
}

function inputText(value) {
  if (value === null || value === undefined) {
    return '';
  }
  if (typeof value === 'string') {
    return value;
  }
  if (Array.isArray(value)) {
    return value.map(inputText).join('\n');
  }
  if (typeof value === 'object') {
    if (typeof value.text === 'string') {
      return value.text;
    }
    if (typeof value.content === 'string') {
      return value.content;
    }
    if (Array.isArray(value.content)) {
      return value.content.map(inputText).join('\n');
    }
    if (Array.isArray(value.parts)) {
      return value.parts.map(inputText).join('\n');
    }
    if (typeof value.message === 'string') {
      return value.message;
    }
    return JSON.stringify(value);
  }
  return String(value);
}

function messagesText(messages) {
  if (!Array.isArray(messages)) {
    return inputText(messages);
  }
  return messages.map(message => {
    const role = message && message.role ? message.role : 'user';
    const content = inputText(message && message.content);
    return `${role}: ${content}`;
  }).join('\n');
}

function completionText(body, family) {
  const source = body.input !== undefined ? inputText(body.input) : messagesText(body.messages || body.contents || body.prompt || '');
  const normalized = source.trim().replace(/\s+/g, ' ');
  const preview = normalized ? normalized.slice(0, 180) : 'empty prompt';
  return `[${family}] deterministic fake response for: ${preview}`;
}

function splitTextChunks(text) {
  const value = String(text);
  if (value.length < 3) {
    return [value];
  }
  const first = Math.max(1, Math.floor(value.length / 3));
  const second = Math.max(first + 1, Math.floor((value.length * 2) / 3));
  return [value.slice(0, first), value.slice(first, second), value.slice(second)].filter(Boolean);
}

function embeddingVector(text, dimensions) {
  const size = Math.max(1, Math.min(Number(dimensions || 16), 3072));
  const seed = crypto.createHash('sha256').update(String(text)).digest();
  const vector = [];
  for (let i = 0; i < size; i += 1) {
    const byte = seed[i % seed.length];
    vector.push(Number(((byte / 255) * 2 - 1).toFixed(6)));
  }
  return vector;
}

function openAiModelsResponse(res) {
  jsonResponse(res, 200, { object: 'list', data: openAiModels });
}

function openAiResponsesResponse(res, body) {
  const text = completionText(body, 'openai.responses');
  const response = {
    id: stableId('resp', body),
    object: 'response',
    created_at: 1_700_000_000,
    status: 'completed',
    model: body.model || 'fake-gpt-4.1',
    output: [
      {
        id: stableId('msg', text),
        type: 'message',
        status: 'completed',
        role: 'assistant',
        content: [
          { type: 'output_text', text, annotations: [] }
        ]
      }
    ],
    output_text: text,
    usage: {
      input_tokens: Math.max(1, inputText(body.input || body.messages || '').length / 4 | 0),
      output_tokens: Math.max(1, text.length / 4 | 0),
      total_tokens: Math.max(2, (inputText(body.input || body.messages || '').length + text.length) / 4 | 0)
    }
  };

  if (body.stream) {
    sseResponse(res, [
      { type: 'response.created', response: { id: response.id, status: 'in_progress' } },
      ...splitTextChunks(text).map(delta => ({ type: 'response.output_text.delta', delta })),
      { type: 'response.completed', response }
    ]);
    return;
  }
  jsonResponse(res, 200, response);
}

function openAiChatResponse(res, body) {
  const text = completionText(body, 'openai.chat');
  const chunkId = stableId('chatcmpl', body);
  if (body.stream) {
    const chunks = splitTextChunks(text).map((content, index) => ({
      id: chunkId,
      object: 'chat.completion.chunk',
      created: 1_700_000_000,
      model: body.model || 'fake-gpt-4.1-mini',
      choices: [{ index: 0, delta: index === 0 ? { role: 'assistant', content } : { content }, finish_reason: null }]
    }));
    chunks.push({
      id: chunkId,
      object: 'chat.completion.chunk',
      created: 1_700_000_000,
      model: body.model || 'fake-gpt-4.1-mini',
      choices: [{ index: 0, delta: {}, finish_reason: 'stop' }],
      usage: {
        prompt_tokens: Math.max(1, messagesText(body.messages).length / 4 | 0),
        completion_tokens: Math.max(1, text.length / 4 | 0),
        total_tokens: Math.max(2, (messagesText(body.messages).length + text.length) / 4 | 0)
      }
    });
    sseResponse(res, chunks);
    return;
  }

  jsonResponse(res, 200, {
    id: chunkId,
    object: 'chat.completion',
    created: 1_700_000_000,
    model: body.model || 'fake-gpt-4.1-mini',
    choices: [
      {
        index: 0,
        message: { role: 'assistant', content: text },
        finish_reason: 'stop'
      }
    ],
    usage: {
      prompt_tokens: Math.max(1, messagesText(body.messages).length / 4 | 0),
      completion_tokens: Math.max(1, text.length / 4 | 0),
      total_tokens: Math.max(2, (messagesText(body.messages).length + text.length) / 4 | 0)
    }
  });
}

function openAiEmbeddingsResponse(res, body) {
  const input = Array.isArray(body.input) ? body.input : [body.input || ''];
  jsonResponse(res, 200, {
    object: 'list',
    model: body.model || 'fake-embedding-3-small',
    data: input.map((item, index) => ({
      object: 'embedding',
      index,
      embedding: embeddingVector(inputText(item), body.dimensions)
    })),
    usage: {
      prompt_tokens: Math.max(1, input.map(inputText).join('\n').length / 4 | 0),
      total_tokens: Math.max(1, input.map(inputText).join('\n').length / 4 | 0)
    }
  });
}

function anthropicModelsResponse(res) {
  jsonResponse(res, 200, {
    data: anthropicModels,
    has_more: false,
    first_id: anthropicModels[0].id,
    last_id: anthropicModels[anthropicModels.length - 1].id
  });
}

function anthropicMessagesResponse(res, body) {
  const text = completionText(body, 'anthropic.messages');
  const response = {
    id: stableId('msg', body),
    type: 'message',
    role: 'assistant',
    model: body.model || 'fake-claude-3-haiku',
    content: [{ type: 'text', text }],
    stop_reason: 'end_turn',
    stop_sequence: null,
    usage: {
      input_tokens: Math.max(1, messagesText(body.messages).length / 4 | 0),
      output_tokens: Math.max(1, text.length / 4 | 0)
    }
  };

  if (body.stream) {
    sseResponse(res, [
      { type: 'message_start', message: { ...response, content: [] } },
      { type: 'content_block_start', index: 0, content_block: { type: 'text', text: '' } },
      ...splitTextChunks(text).map(chunk => ({ type: 'content_block_delta', index: 0, delta: { type: 'text_delta', text: chunk } })),
      { type: 'content_block_stop', index: 0 },
      { type: 'message_delta', delta: { stop_reason: 'end_turn', stop_sequence: null }, usage: response.usage },
      { type: 'message_stop' }
    ]);
    return;
  }

  jsonResponse(res, 200, response);
}

function geminiModelsResponse(res) {
  jsonResponse(res, 200, { models: geminiModels });
}

function geminiContentText(contents) {
  if (!Array.isArray(contents)) {
    return inputText(contents);
  }
  return contents.map(content => inputText(content && content.parts ? content.parts : content)).join('\n');
}

function geminiGenerateContentResponse(res, body, modelName, stream) {
  const text = completionText({ ...body, contents: geminiContentText(body.contents) }, 'gemini.generateContent');
  const response = {
    candidates: [
      {
        index: 0,
        content: {
          role: 'model',
          parts: [{ text }]
        },
        finishReason: 'STOP'
      }
    ],
    usageMetadata: {
      promptTokenCount: Math.max(1, geminiContentText(body.contents).length / 4 | 0),
      candidatesTokenCount: Math.max(1, text.length / 4 | 0),
      totalTokenCount: Math.max(2, (geminiContentText(body.contents).length + text.length) / 4 | 0)
    },
    modelVersion: modelName || 'models/fake-gemini-1.5-pro'
  };

  if (stream) {
    const chunks = splitTextChunks(text).map((chunk, index, items) => ({
      candidates: [{
        index: 0,
        content: { role: 'model', parts: [{ text: chunk }] },
        finishReason: index === items.length - 1 ? 'STOP' : undefined
      }],
      usageMetadata: index === items.length - 1 ? response.usageMetadata : undefined,
      modelVersion: response.modelVersion
    }));
    sseResponse(res, chunks);
    return;
  }
  jsonResponse(res, 200, response);
}

function geminiEmbedResponse(res, body) {
  const content = body.content || (Array.isArray(body.requests) && body.requests[0] && body.requests[0].content) || '';
  const text = inputText(content.parts || content);
  jsonResponse(res, 200, {
    embedding: {
      values: embeddingVector(text, body.outputDimensionality || 16)
    }
  });
}

function geminiBatchEmbedResponse(res, body) {
  const requests = Array.isArray(body.requests) ? body.requests : [];
  jsonResponse(res, 200, {
    embeddings: requests.map(request => ({
      values: embeddingVector(inputText(request.content && request.content.parts ? request.content.parts : request.content), request.outputDimensionality || body.outputDimensionality || 16)
    }))
  });
}

function geminiModelRoute(path) {
  const match = path.match(/^\/(?:gemini\/)?(?:v1beta\/)?models\/(.+):(generateContent|streamGenerateContent|embedContent|batchEmbedContents)$/);
  if (!match) {
    return null;
  }
  return {
    modelName: `models/${match[1]}`,
    method: match[2]
  };
}

async function handleRequest(req, res) {
  const url = new URL(req.url, `http://${req.headers.host || `${host}:${port}`}`);
  const path = url.pathname.replace(/\/+$/, '') || '/';

  res.setHeader('access-control-allow-origin', '*');
  res.setHeader('access-control-allow-headers', 'content-type, authorization, x-api-key, anthropic-version');
  res.setHeader('access-control-allow-methods', 'GET, POST, OPTIONS');

  if (req.method === 'OPTIONS') {
    textResponse(res, 204, '');
    return;
  }

  if (req.method === 'GET' && (path === '/' || path === '/health')) {
    jsonResponse(res, 200, { status: 'ok', provider: 'fake-provider' });
    return;
  }

  if (req.method === 'GET' && (path === '/v1/models' || path === '/openai/v1/models')) {
    openAiModelsResponse(res);
    return;
  }

  if (req.method === 'GET' && (path === '/anthropic/v1/models' || path === '/anthropic/models' || path === '/v1/anthropic/models')) {
    anthropicModelsResponse(res);
    return;
  }

  if (req.method === 'GET' && (path === '/gemini/v1beta/models' || path === '/gemini/models' || path === '/v1beta/models')) {
    geminiModelsResponse(res);
    return;
  }

  if (req.method !== 'POST') {
    methodNotAllowed(res, req.method);
    return;
  }

  let body;
  try {
    body = await readBody(req);
  } catch (error) {
    jsonResponse(res, 400, { error: { type: 'invalid_request_error', message: error.message } });
    return;
  }

  if (path === '/v1/responses' || path === '/openai/v1/responses') {
    openAiResponsesResponse(res, body);
    return;
  }

  if (path === '/v1/chat/completions' || path === '/openai/v1/chat/completions') {
    openAiChatResponse(res, body);
    return;
  }

  if (path === '/v1/embeddings' || path === '/openai/v1/embeddings') {
    openAiEmbeddingsResponse(res, body);
    return;
  }

  if (path === '/anthropic/v1/messages' || path === '/anthropic/messages' || path === '/v1/anthropic/messages') {
    anthropicMessagesResponse(res, body);
    return;
  }

  const geminiRoute = geminiModelRoute(path);
  if (geminiRoute) {
    if (geminiRoute.method === 'generateContent' || geminiRoute.method === 'streamGenerateContent') {
      geminiGenerateContentResponse(res, body, geminiRoute.modelName, geminiRoute.method === 'streamGenerateContent');
      return;
    }
    if (geminiRoute.method === 'batchEmbedContents') {
      geminiBatchEmbedResponse(res, body);
      return;
    }
    geminiEmbedResponse(res, body);
    return;
  }

  notFound(res, path);
}

const server = http.createServer((req, res) => {
  handleRequest(req, res).catch(error => {
    console.error('[fake-provider] unhandled request error', error);
    jsonResponse(res, 500, { error: { type: 'internal_error', message: error.message } });
  });
});

server.listen(port, host, () => {
  console.log(`[fake-provider] listening on http://${host}:${port}`);
});

function shutdown(signal) {
  console.log(`[fake-provider] received ${signal}, shutting down`);
  server.close(error => {
    if (error) {
      console.error('[fake-provider] shutdown failed', error);
      process.exit(1);
    }
    process.exit(0);
  });
}

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);
