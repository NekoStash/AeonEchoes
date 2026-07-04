<script setup lang="ts">
import { Bot, BrainCircuit, FileClock, Lightbulb, Loader2, Save, Sparkles } from '@lucide/vue'
import type {
  AgentConfig,
  AgentRunResult,
  ChapterVersion,
  ContinuityAudit,
  ContinuityIssue,
  ContextPack,
  ContextPreviewResponse,
  ContextPreviewSummary,
  ContextSelection,
  IndexFreshness,
  IndexJob,
  ModelResolution,
  StoryBible,
  ToolTrace
} from '~/lib/types'
import { cn, formatDateTime } from '~/lib/utils'

type PreviewTarget = 'chapter_idea' | 'draft'
type ReferenceSelectionState = {
  include_chapter_plan: boolean
  include_chapter_summary: boolean
  include_world_rules: boolean
  previous_chapter_count: number
  character_ids: string[]
}

type StoryCharacter = StoryBible['characters'][number]

const { t } = useI18n()
const route = useRoute()
const projectId = computed(() => String(route.params.id))
const routeChapterId = computed(() => {
  const value = route.query.chapter
  if (Array.isArray(value)) return String(value[0] || '')
  return String(value || '')
})
const api = useApi()
const workspace = useWorkspaceStore()
const selectedChapterId = ref(routeChapterId.value)
const chapterId = computed(() => selectedChapterId.value || routeChapterId.value || '')

const title = ref(t('editor.defaults.title'))
const content = ref(t('editor.defaults.content'))
const prompt = ref(t('editor.defaults.prompt'))
const chapterPlan = ref(t('editor.defaults.chapterPlan'))
const chapterIdeaWorkflowId = ref('')
const styleConstraints = ref(t('editor.defaults.styleConstraints'))
const versions = ref<ChapterVersion[]>([])
const agentRunResult = ref<AgentRunResult | null>(null)
const agents = ref<AgentConfig[]>([])
const loadingAgents = ref(false)
const loadingVersions = ref(false)
const planning = ref(false)
const drafting = ref(false)
const savingVersion = ref(false)
const previewLoading = ref(false)
const previewError = ref('')
const previewResult = ref<ContextPreviewResponse | null>(null)
const previewTarget = ref<PreviewTarget | null>(null)
const selectedVersionId = ref('')
const loadedVersionId = ref('')
const planStatus = ref<'idle' | 'saved'>('idle')
const draftStatus = ref<'idle' | 'appended'>('idle')
const saveStatus = ref<'idle' | 'saved'>('idle')
const localError = ref('')
const activePanel = ref('reference')
const editorMainRef = ref<HTMLElement | { $el?: HTMLElement } | null>(null)
const sidePanelRef = ref<HTMLElement | { $el?: HTMLElement } | null>(null)
const previewResultRef = ref<HTMLElement | null>(null)
const referenceSelection = reactive<ReferenceSelectionState>({
  include_chapter_plan: false,
  include_chapter_summary: true,
  include_world_rules: false,
  previous_chapter_count: 0,
  character_ids: []
})

const tabs = computed(() => [
  { label: t('editor.context'), value: 'reference' },
  { label: t('editor.plan'), value: 'plan' },
  { label: t('editor.draft'), value: 'draft' },
  { label: t('editor.review'), value: 'review' },
  { label: t('editor.versions'), value: 'versions', badge: String(versions.value.length) },
  { label: t('editor.runLog'), value: 'runs', badge: String(projectIndexJobs.value.length) }
])

const availableChapters = computed(() => workspace.activeChapters)
const chapterOptions = computed(() => availableChapters.value.map((chapter) => ({
  label: chapter.title || t('editor.chapterFallbackTitle', { number: chapter.number || availableChapters.value.findIndex((item) => item.id === chapter.id) + 1 }),
  value: chapter.id,
  description: t('editor.chapterOptionDescription', { number: chapter.number || availableChapters.value.findIndex((item) => item.id === chapter.id) + 1, status: chapterStatusLabel(chapter.status) })
})))
const currentChapter = computed(() => availableChapters.value.find((chapter) => chapter.id === chapterId.value))
const currentChapterIndex = computed(() => availableChapters.value.findIndex((chapter) => chapter.id === chapterId.value))
const hasRealCurrentChapter = computed(() => Boolean(currentChapter.value?.id && currentChapter.value.id === chapterId.value))
const hasInvalidRouteChapter = computed(() => Boolean(routeChapterId.value && availableChapters.value.length > 0 && !availableChapters.value.some((chapter) => chapter.id === routeChapterId.value)))
const chapterMetaLabel = computed(() => hasInvalidRouteChapter.value ? t('editor.invalidChapterLabel', { id: routeChapterId.value }) : chapterId.value)
const previousChapterOptions = computed(() => availableChapters.value.slice(0, Math.max(currentChapterIndex.value, 0)))
const selectedPreviousChapters = computed(() => {
  const count = Math.max(0, Number(referenceSelection.previous_chapter_count || 0))
  if (count === 0 || currentChapterIndex.value <= 0) return []
  return availableChapters.value.slice(Math.max(0, currentChapterIndex.value - count), currentChapterIndex.value)
})
const selectedContextChapterIds = computed(() => uniqueTrimmed([
  ...selectedPreviousChapters.value.map((chapter) => chapter.id),
  ...(referenceSelection.include_chapter_summary && hasRealCurrentChapter.value && currentChapter.value?.id ? [currentChapter.value.id] : [])
]))
const availableCharacters = computed(() => (workspace.activeBible?.characters || []).filter((character) => character.name.trim()))
const selectedCharacterIdSet = computed(() => new Set(referenceSelection.character_ids))
const selectedCharacters = computed(() => availableCharacters.value.filter((character) => selectedCharacterIdSet.value.has(character.id)))
const hasManualCharacterSelection = computed(() => referenceSelection.character_ids.length > 0)
const autoSelectedCharacters = computed(() => inferAutoSelectedCharacters())
const requestCharacters = computed(() => (hasManualCharacterSelection.value ? selectedCharacters.value : autoSelectedCharacters.value))
const requestCharacterEntityIds = computed(() => uniqueTrimmed(requestCharacters.value.map((character) => character.entity_id || '')).slice(0, 3))
const unsyncedRequestCharacters = computed(() => requestCharacters.value.filter((character) => !character.entity_id?.trim()))
const hasUnsyncedRequestCharacters = computed(() => unsyncedRequestCharacters.value.length > 0)
const unsyncedRequestCharacterNames = computed(() => uniqueTrimmed(unsyncedRequestCharacters.value.map((character) => character.name)))
const hasReferenceFocus = computed(() => Boolean(
  referenceSelection.include_chapter_plan
  || (referenceSelection.include_chapter_summary && hasRealCurrentChapter.value)
  || referenceSelection.previous_chapter_count > 0
  || referenceSelection.include_world_rules
  || hasManualCharacterSelection.value
))
const selectionPayload = computed<ContextSelection>(() => ({
  chapter_ids: selectedContextChapterIds.value.length ? selectedContextChapterIds.value : undefined,
  previous_chapter_count: referenceSelection.previous_chapter_count > 0 ? referenceSelection.previous_chapter_count : undefined,
  include_current_chapter: referenceSelection.include_chapter_summary || undefined,
  include_world_rules: referenceSelection.include_world_rules || undefined,
  character_ids: requestCharacterEntityIds.value.length ? requestCharacterEntityIds.value : undefined
}))
const referenceFocusChips = computed(() => {
  const chips: string[] = []
  if (referenceSelection.include_chapter_plan && chapterPlan.value.trim()) chips.push(t('editor.referenceFocusLabels.chapterPlan'))
  if (referenceSelection.previous_chapter_count > 0) chips.push(t('editor.referenceFocusLabels.previousChapters', { count: selectedPreviousChapters.value.length }))
  if (referenceSelection.include_chapter_summary && currentChapter.value?.summary.trim()) chips.push(t('editor.referenceFocusLabels.chapterSummary'))
  if (referenceSelection.include_world_rules && workspace.activeBible?.world_rules.some((rule) => rule.trim())) chips.push(t('editor.referenceFocusLabels.worldRules'))
  selectedCharacters.value.forEach((character) => chips.push(character.name.trim()))
  if (!hasManualCharacterSelection.value) autoSelectedCharacters.value.forEach((character) => chips.push(t('editor.autoCharacters.chip', { name: character.name.trim() })))
  return chips
})

const outlineCards = computed(() => {
  const chapter = currentChapter.value
  return [
    { title: t('editor.outline.currentChapter.title'), description: chapter?.title || t('editor.outline.currentChapter.description') },
    { title: t('editor.outline.chapterGoal.title'), description: chapter?.summary || t('editor.outline.chapterGoal.description') },
    { title: t('editor.outline.nextAction.title'), description: t('editor.outline.nextAction.description') }
  ]
})

const metrics = computed(() => {
  const wordCount = content.value.replace(/\s/g, '').length
  return {
    wordCount,
    paragraphs: content.value.split(/\n{2,}/).filter(Boolean).length
  }
})

const referenceSections = computed(() => buildReferenceSections(
  workspace.activeBible,
  currentChapter.value?.id,
  chapterPlan.value,
  referenceSelection,
  hasReferenceFocus.value
))

const previewSummary = computed<ContextPreviewSummary | null>(() => {
  if (!previewResult.value) return null
  return buildContextPreviewSummary(previewResult.value.context_pack)
})

const previewContextSections = computed(() => buildPreviewContextSections(previewResult.value?.context_pack))
const previewStructuredContext = computed(() => previewContextSections.value.length > 0)
const previewTargetLabelValue = computed(() => (previewTarget.value ? previewTargetLabel(previewTarget.value) : t('common.emptyValue')))
const currentToolTrace = computed(() => normalizeTraceItems(
  agentRunResult.value?.tool_trace
  || previewResult.value?.tool_trace
  || previewResult.value?.context_pack?.tool_trace
  || []
))
const previewToolTrace = computed(() => normalizeTraceItems(previewResult.value?.tool_trace || previewResult.value?.context_pack?.tool_trace || []))
const toolTraceSummary = computed(() => buildToolTraceSummary(currentToolTrace.value))
const previewToolTraceSummary = computed(() => buildToolTraceSummary(previewToolTrace.value))

const workflowSteps = computed(() => {
  if (agentRunResult.value?.run) {
    return [
      { id: 'agent-run', name: t('editor.workflow.agentRun.name'), status: agentRunResult.value.run.status === 'completed' ? 'completed' as const : agentRunResult.value.run.status === 'failed' ? 'failed' as const : 'running' as const, message: agentRunResult.value.run.id },
      { id: 'agent-output', name: t('editor.workflow.agentOutput.name'), status: agentRunResult.value.content ? 'completed' as const : 'idle' as const, message: agentRunResult.value.content ? t('editor.workflow.agentOutput.message') : t('editor.workflow.context.message') }
    ]
  }
  return [
    { id: 'idle-context', name: t('editor.workflow.context.name'), status: 'idle' as const, message: t('editor.workflow.context.message') },
    { id: 'idle-plan', name: t('editor.workflow.plan.name'), status: 'idle' as const, message: t('editor.workflow.plan.message') },
    { id: 'idle-draft', name: t('editor.workflow.draft.name'), status: 'idle' as const, message: t('editor.workflow.draft.message') },
    { id: 'idle-review', name: t('editor.workflow.review.name'), status: 'idle' as const, message: t('editor.workflow.review.message') }
  ]
})

const projectIndexJobs = computed(() => workspace.indexJobs.filter((item) => item.project_id === projectId.value))
const latestIndexJobs = computed(() => projectIndexJobs.value.slice(0, 5))
const indexJobsLoading = computed(() => Object.entries(workspace.loading).some(([key, value]) => key.startsWith(`index-jobs:${projectId.value}`) && value))
const backgroundIndexState = computed(() => {
  const jobs = projectIndexJobs.value
  if (indexJobsLoading.value) return { variant: 'gold' as const, label: t('editor.backgroundIndex.loading') }
  if (jobs.some((item) => item.status === 'running')) return { variant: 'gold' as const, label: t('editor.backgroundIndex.running') }
  if (jobs.some((item) => item.status === 'failed')) return { variant: 'rose' as const, label: t('editor.backgroundIndex.failed') }
  if (jobs.some((item) => item.status === 'pending')) return { variant: 'gold' as const, label: t('editor.backgroundIndex.pending') }
  if (jobs.some((item) => item.status === 'superseded')) return { variant: 'muted' as const, label: t('editor.backgroundIndex.superseded') }
  if (jobs.some((item) => item.status === 'completed')) return { variant: 'success' as const, label: t('editor.backgroundIndex.completed') }
  return { variant: 'muted' as const, label: t('editor.backgroundIndex.idle') }
})

const diagnosticsModelResolution = computed<ModelResolution | null>(() => (
  agentRunResult.value?.model_resolution
  || previewResult.value?.model_resolution
  || null
))
const diagnosticsFreshness = computed<IndexFreshness | null>(() => previewResult.value?.index_freshness || null)
const diagnosticsExecutionTarget = computed(() => {
  const taskType = String(agentRunResult.value?.run?.input?.task_type || '')
  if (taskType === 'draft') return t('editor.diagnostics.executionTargets.draft')
  if (taskType === 'chapter_idea') return t('editor.diagnostics.executionTargets.chapterIdea')
  if (previewTarget.value) return previewTargetLabel(previewTarget.value)
  return t('editor.diagnostics.executionTargets.idle')
})
const draftContinuityAudit = computed<ContinuityAudit | null>(() => null)
const continuityAuditIssues = computed<ContinuityIssue[]>(() => [])
const continuityAuditPassed = computed(() => false)
const draftWarnings = computed(() => [])

watch(availableCharacters, (characters) => {
  const validIds = new Set(characters.map((character) => character.id))
  referenceSelection.character_ids = referenceSelection.character_ids.filter((id) => validIds.has(id))
}, { immediate: true })

onMounted(async () => {
  await workspace.loadStoryBible(projectId.value)
  initializeSelectedChapter()
  await loadAgents()
  await loadVersions()
  await refreshIndexJobs()
})

watch(routeChapterId, (id) => {
  if (id && id !== selectedChapterId.value) {
    selectedChapterId.value = id
  }
})

watch(availableChapters, () => {
  initializeSelectedChapter()
  clampPreviousChapterCount()
}, { immediate: true })

watch(chapterId, async (id, oldId) => {
  if (!id || id === oldId) return
  await loadVersions()
  await refreshIndexJobs()
})

watch(() => referenceSelection.previous_chapter_count, () => clampPreviousChapterCount())

function uniqueTrimmed(values: string[]) {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean)))
}

function initializeSelectedChapter() {
  if (selectedChapterId.value) return
  const firstChapter = availableChapters.value[0]
  if (firstChapter?.id) selectedChapterId.value = firstChapter.id
}

function clampPreviousChapterCount() {
  const maxCount = Math.max(currentChapterIndex.value, 0)
  const nextCount = Math.min(Math.max(0, Number(referenceSelection.previous_chapter_count || 0)), maxCount)
  if (referenceSelection.previous_chapter_count !== nextCount) {
    referenceSelection.previous_chapter_count = nextCount
  }
}

async function replaceChapterRoute(id: string) {
  selectedChapterId.value = id
  await navigateTo({ path: route.path, query: { ...route.query, chapter: id, keepLocal: undefined } }, { replace: true })
}

async function selectChapter(id: string) {
  if (!id || id === chapterId.value) return
  await replaceChapterRoute(id)
}

function ensureNumberFromCurrentSelection() {
  if (currentChapter.value?.number) return currentChapter.value.number
  const existingIndex = availableChapters.value.findIndex((chapter) => chapter.id === chapterId.value)
  if (existingIndex >= 0) return existingIndex + 1
  const routeMatch = chapterId.value.match(/(\d+)$/)
  if (routeMatch) return Number(routeMatch[1])
  return availableChapters.value.length + 1
}

async function ensureCurrentChapter() {
  const ensured = await workspace.ensureChapter(projectId.value, {
    chapter_id: chapterId.value || undefined,
    number: ensureNumberFromCurrentSelection(),
    title: title.value.trim() || currentChapter.value?.title || undefined,
    status: currentChapter.value?.status || 'drafting',
    summary: currentChapter.value?.summary || prompt.value.trim() || undefined,
    metadata: currentChapter.value?.metadata
  })
  const realChapterId = ensured.data.chapter.id
  if (realChapterId && realChapterId !== chapterId.value) {
    await replaceChapterRoute(realChapterId)
  } else if (realChapterId) {
    selectedChapterId.value = realChapterId
  }
  return realChapterId || chapterId.value
}

function includesText(source: string, needle: string) {
  const trimmedNeedle = needle.trim()
  return Boolean(trimmedNeedle) && source.includes(trimmedNeedle)
}

function isProtagonist(character: StoryCharacter, index: number) {
  const role = character.role.trim().toLowerCase()
  return index === 0 || role.includes('主角') || role.includes('protagonist') || role.includes('lead')
}

function inferAutoSelectedCharacters() {
  const characters = availableCharacters.value
  if (characters.length === 0) return []

  const sourceText = [
    currentChapter.value?.title,
    currentChapter.value?.summary,
    chapterPlan.value,
    prompt.value,
    title.value
  ].map((item) => item?.trim()).filter(Boolean).join('\n')
  const matched = characters.filter((character) => includesText(sourceText, character.name.trim()))
  const protagonist = characters.find((character, index) => isProtagonist(character, index)) || characters[0]
  const ordered = matched.length ? matched : protagonist ? [protagonist] : []
  if (matched.length && protagonist && !ordered.some((character) => character.id === protagonist.id)) ordered.unshift(protagonist)

  return ordered.slice(0, 3)
}

function characterAnchorLabel(character: StoryCharacter) {
  const details = [
    character.role ? t('editor.characterAnchor.role', { value: character.role.trim() }) : '',
    character.desire ? t('editor.characterAnchor.desire', { value: character.desire.trim() }) : '',
    character.wound ? t('editor.characterAnchor.wound', { value: character.wound.trim() }) : '',
    character.secret ? t('editor.characterAnchor.secret', { value: character.secret.trim() }) : ''
  ].filter(Boolean)
  return details.length ? `${character.name.trim()}（${details.join(t('common.listSeparator'))}）` : character.name.trim()
}

function selectedCharacterAnchorLine() {
  const anchors = requestCharacters.value.map(characterAnchorLabel).filter(Boolean)
  if (anchors.length === 0) return ''
  return hasManualCharacterSelection.value
    ? t('editor.selectionBrief.characters', { content: anchors.join('；') })
    : t('editor.selectionBrief.autoCharacters', { content: anchors.join('；') })
}

function buildReferenceSections(
  bible: StoryBible | null,
  activeChapterId: string | undefined,
  currentChapterPlan: string,
  selection: ReferenceSelectionState,
  hasFocusedScope: boolean
) {
  if (!bible) return []

  const selectedCharacterIDs = new Set(selection.character_ids || [])
  const highlightedKeys = new Set<string>()
  if (selection.include_chapter_plan) highlightedKeys.add('chapter-plan')
  if (selection.include_chapter_summary || selection.previous_chapter_count > 0) highlightedKeys.add('chapter')
  if (selection.include_world_rules) highlightedKeys.add('rules')
  if (selectedCharacterIDs.size > 0) highlightedKeys.add('characters')

  const selectedChapterIds = new Set(selectedContextChapterIds.value)
  const chapterSummaryItems = bible.chapters
    .filter((chapter) => (selectedChapterIds.has(chapter.id) || (!hasFocusedScope && chapter.id === activeChapterId)) && chapter.summary.trim())
    .map((chapter) => `${chapter.title || chapter.id}：${chapter.summary.trim()}`)

  const allCharacters = bible.characters
    .map((character) => {
      const parts = [character.name, character.role, character.desire].map((item) => item?.trim()).filter(Boolean)
      return { id: character.id, label: parts.join(' · ') }
    })
    .filter((character) => character.label)

  const sections = [
    {
      key: 'chapter-plan',
      title: t('editor.referenceSections.chapterPlan'),
      emphasized: highlightedKeys.has('chapter-plan'),
      items: selection.include_chapter_plan ? [currentChapterPlan.trim()].filter(Boolean) : []
    },
    {
      key: 'chapter',
      title: t('editor.referenceSections.chapterSummary'),
      emphasized: highlightedKeys.has('chapter'),
      items: selection.include_chapter_summary || !hasFocusedScope ? chapterSummaryItems : []
    },
    {
      key: 'rules',
      title: t('editor.referenceSections.worldRules'),
      emphasized: highlightedKeys.has('rules'),
      items: selection.include_world_rules || !hasFocusedScope ? bible.world_rules.map((rule) => rule.trim()).filter(Boolean) : []
    },
    {
      key: 'characters',
      title: t('editor.referenceSections.characters'),
      emphasized: highlightedKeys.has('characters'),
      items: selectedCharacterIDs.size > 0
        ? allCharacters.filter((character) => selectedCharacterIDs.has(character.id)).map((character) => character.label)
        : !hasFocusedScope
          ? allCharacters.map((character) => character.label)
          : []
    },
    {
      key: 'premise',
      title: t('editor.referenceSections.premise'),
      emphasized: false,
      items: [bible.premise].filter(Boolean)
    },
    {
      key: 'foreshadows',
      title: t('editor.referenceSections.foreshadows'),
      emphasized: false,
      items: bible.foreshadows.map((item) => {
        const parts = [item.title, item.payoff_hint].map((value) => value?.trim()).filter(Boolean)
        return parts.join(' · ')
      }).filter(Boolean)
    }
  ]

  return sections.filter((section) => section.items.length > 0)
}

function buildPreviewContextSections(contextPack?: ContextPack) {
  if (!contextPack) return []
  return [
    {
      key: 'chapter_summaries',
      title: t('editor.preview.sections.chapterSummaries'),
      items: (contextPack.chapter_summaries || []).map((item) => `${item.title} · ${item.summary}`)
    },
    {
      key: 'world_rules',
      title: t('editor.preview.sections.worldRules'),
      items: Object.values(contextPack.world_rules || {}).map((item) => item.trim()).filter(Boolean)
    },
    {
      key: 'entities',
      title: t('editor.preview.sections.entities'),
      items: (contextPack.entities || []).map((entity) => `${entity.name} · ${entity.summary}`)
    },
    {
      key: 'facts',
      title: t('editor.preview.sections.facts'),
      items: (contextPack.facts || []).map((fact) => fact.claim.trim()).filter(Boolean)
    },
    {
      key: 'plot_threads',
      title: t('editor.preview.sections.plotThreads'),
      items: (contextPack.plot_threads || []).map((thread) => [thread.title, thread.summary].map((item) => item.trim()).filter(Boolean).join(' · ')).filter(Boolean)
    }
  ].filter((section) => section.items.length > 0)
}

function normalizeTraceItems(trace: ToolTrace[]) {
  return trace.map((item, index) => {
    if (typeof item === 'string') {
      const [rawName, ...rest] = item.split(/\s+/)
      return {
        id: `trace-${index}`,
        raw: item,
        name: rawName || t('editor.toolTrace.unknownTool'),
        category: inferToolTraceCategory(item),
        summary: rest.join(' ') || item
      }
    }
    const name = item.tool || item.name || t('editor.toolTrace.unknownTool')
    const values = [
      item.status,
      item.chapter_id,
      item.chapter_ids?.join(', '),
      item.character_id,
      item.character_ids?.join(', '),
      item.entity_id,
      item.entity_ids?.join(', '),
      item.event_id,
      item.event_ids?.join(', '),
      item.timeline === undefined ? '' : String(item.timeline),
      item.count === undefined ? '' : t('editor.toolTrace.count', { count: item.count }),
      item.message
    ].map((value) => value?.trim()).filter(Boolean)
    const valueSummary = values.join(' · ')
    return {
      id: `trace-${index}`,
      raw: JSON.stringify(item),
      name,
      category: inferToolTraceCategory([name, ...values].join(' ')),
      summary: valueSummary || (item.metadata ? JSON.stringify(item.metadata) : name)
    }
  })
}

function inferToolTraceCategory(value: string) {
  const normalized = value.toLowerCase()
  if (normalized.includes('chapter')) return 'chapters'
  if (normalized.includes('character') || normalized.includes('entity') || normalized.includes('graph')) return 'characters'
  if (normalized.includes('event') || normalized.includes('fact')) return 'events'
  if (normalized.includes('timeline') || normalized.includes('time')) return 'timeline'
  return 'other'
}

function buildToolTraceSummary(trace: ReturnType<typeof normalizeTraceItems>) {
  const counts = trace.reduce<Record<string, number>>((items, item) => {
    items[item.category] = (items[item.category] || 0) + 1
    return items
  }, {})
  return {
    items: trace,
    counts,
    hasTrace: trace.length > 0,
    text: t('editor.toolTrace.summary', {
      chapters: counts.chapters || 0,
      characters: counts.characters || 0,
      events: counts.events || 0,
      timeline: counts.timeline || 0,
      other: counts.other || 0
    })
  }
}

function toolTraceCategoryLabel(category: string) {
  return translatedStatusOrFallback('editor.toolTrace.categories', category)
}

function buildContextPreviewSummary(contextPack: ContextPack): ContextPreviewSummary {
  const chapterSummaryCount = contextPack.chapter_summaries?.length || 0
  const entityCount = contextPack.entities?.length || 0
  const factCount = contextPack.facts?.length || 0
  const plotThreadCount = contextPack.plot_threads?.length || 0
  const worldRuleCount = Object.keys(contextPack.world_rules || {}).length
  return {
    chapter_summary_count: chapterSummaryCount,
    entity_count: entityCount,
    fact_count: factCount,
    plot_thread_count: plotThreadCount,
    world_rule_count: worldRuleCount,
    text: t('editor.preview.summaryText', {
      chapterSummaryCount,
      entityCount,
      factCount,
      plotThreadCount,
      worldRuleCount
    })
  }
}

function resetChapterState() {
  const chapter = currentChapter.value
  title.value = chapter?.title || t('editor.defaults.titleForChapter', { id: chapterId.value })
  content.value = chapter?.summary ? `${chapter.summary}\n\n${t('editor.defaults.emptyContent')}` : t('editor.defaults.emptyContent')
  prompt.value = chapter?.summary || t('editor.defaults.prompt')
  chapterPlan.value = chapter?.summary ? t('editor.defaults.chapterPlanFromSummary', { summary: chapter.summary }) : t('editor.defaults.chapterPlan')
  agentRunResult.value = null
  versions.value = []
  chapterIdeaWorkflowId.value = ''
  previewError.value = ''
  previewResult.value = null
  previewTarget.value = null
  selectedVersionId.value = ''
  loadedVersionId.value = ''
  planStatus.value = 'idle'
  draftStatus.value = 'idle'
  saveStatus.value = 'idle'
  activePanel.value = 'reference'
}

function resolveElement(element: HTMLElement | { $el?: HTMLElement } | null) {
  if (!element) return null
  if (element instanceof HTMLElement) return element
  return element.$el || null
}

function focusPanel(element: HTMLElement | { $el?: HTMLElement } | null) {
  const target = resolveElement(element)
  if (!target) return
  target.scrollIntoView({ behavior: 'smooth', block: 'start' })
  target.focus({ preventScroll: true })
}

async function focusMainEditor() {
  await nextTick()
  focusPanel(editorMainRef.value)
}

async function focusSidePanel() {
  await nextTick()
  focusPanel(sidePanelRef.value)
}

async function focusPreviewResult() {
  await nextTick()
  focusPanel(previewResultRef.value)
}

async function loadAgents() {
  loadingAgents.value = true
  try {
    const result = await api.listAgents({ projectId: projectId.value, enabled: true })
    workspace.recordResult(t('editor.resultScopes.agentRuns'), result)
    agents.value = result.data
  } catch (error) {
    const apiError = workspace.recordError(t('editor.resultScopes.agentRuns'), error)
    localError.value = apiError.message || t('editor.errors.loadAgentsFailed')
  } finally {
    loadingAgents.value = false
  }
}

async function resolveAgentForRole(role: AgentConfig['role']) {
  if (agents.value.length === 0) {
    await loadAgents()
  }
  const agent = agents.value.find((item) => item.enabled && item.role === role)
    || agents.value.find((item) => item.enabled && !item.role)
    || agents.value.find((item) => item.enabled)
  if (!agent?.id) {
    throw new Error(t('editor.errors.noAgentConfigured'))
  }
  return agent
}

async function loadVersion(version: ChapterVersion) {
  selectedVersionId.value = version.id
  loadedVersionId.value = version.id
  title.value = version.title
  content.value = version.content
  chapterPlan.value = version.metadata?.chapter_plan || chapterPlan.value
  planStatus.value = 'idle'
  draftStatus.value = 'idle'
  saveStatus.value = 'idle'
  await focusMainEditor()
}

async function loadVersions() {
  localError.value = ''
  loadingVersions.value = true
  resetChapterState()
  try {
    const result = await api.listChapterVersions(projectId.value, chapterId.value)
    workspace.recordResult(t('editor.resultScopes.chapterVersions'), result)
    versions.value = result.data
    const latestVersion = result.data[0]
    if (latestVersion && !route.query.keepLocal) {
      title.value = latestVersion.title
      content.value = latestVersion.content
      chapterPlan.value = latestVersion.metadata?.chapter_plan || chapterPlan.value
      selectedVersionId.value = latestVersion.id
      loadedVersionId.value = latestVersion.id
    }
  } catch (error) {
    const apiError = workspace.recordError(t('editor.resultScopes.chapterVersions'), error)
    localError.value = apiError.message || t('editor.errors.loadVersionsFailed')
  } finally {
    loadingVersions.value = false
  }
}

async function refreshIndexJobs() {
  const result = await workspace.loadIndexJobs(projectId.value)
  if (!result) {
    localError.value = localError.value || t('editor.errors.loadIndexJobsFailed')
  }
}

function styleConstraintList() {
  return styleConstraints.value
    .split(/[，,]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function referenceCharacterLabel(character: StoryBible['characters'][number]) {
  return [character.name, character.role, character.desire].map((item) => item?.trim()).filter(Boolean).join(' · ')
}

function buildSelectedReferenceBriefLines(options?: { includeChapterPlan?: boolean; includeChapterSummary?: boolean }) {
  const lines: string[] = []

  if (options?.includeChapterPlan && referenceSelection.include_chapter_plan && chapterPlan.value.trim()) {
    lines.push(t('editor.selectionBrief.chapterPlan', { content: chapterPlan.value.trim() }))
  }

  if (selectedPreviousChapters.value.length > 0) {
    const previousSummaries = selectedPreviousChapters.value
      .map((chapter) => [chapter.title || chapter.id, chapter.summary].map((item) => item?.trim()).filter(Boolean).join('：'))
      .filter(Boolean)
    if (previousSummaries.length) lines.push(t('editor.selectionBrief.previousChapters', { content: previousSummaries.join('；') }))
  }

  if (options?.includeChapterSummary && referenceSelection.include_chapter_summary && currentChapter.value?.summary.trim()) {
    lines.push(t('editor.selectionBrief.chapterSummary', { content: currentChapter.value.summary.trim() }))
  }

  if (referenceSelection.include_world_rules) {
    const worldRules = workspace.activeBible?.world_rules.map((rule) => rule.trim()).filter(Boolean) || []
    if (worldRules.length) lines.push(t('editor.selectionBrief.worldRules', { content: worldRules.join('；') }))
  }

  const firstRequestCharacter = requestCharacters.value[0]
  if (!hasManualCharacterSelection.value && firstRequestCharacter) {
    lines.push(t('editor.selectionBrief.protagonistPriority', { name: firstRequestCharacter.name.trim() }))
  }

  const characterAnchorLine = selectedCharacterAnchorLine()
  if (characterAnchorLine) lines.push(characterAnchorLine)

  return lines
}

function buildChapterIdeaBrief() {
  return [
    prompt.value,
    !hasReferenceFocus.value && currentChapter.value?.summary ? t('editor.chapterSummaryLine', { summary: currentChapter.value.summary }) : '',
    ...buildSelectedReferenceBriefLines({ includeChapterPlan: true, includeChapterSummary: true })
  ].filter(Boolean).join('\n')
}

function buildDraftBrief() {
  return [
    prompt.value,
    t('editor.chapterPlanLine', { plan: chapterPlan.value }),
    ...buildSelectedReferenceBriefLines({ includeChapterPlan: false, includeChapterSummary: true })
  ].filter(Boolean).join('\n')
}

function previewTargetLabel(target: PreviewTarget) {
  return t(`editor.preview.targets.${target}`)
}

function toggleCharacterReference(characterId: string) {
  const index = referenceSelection.character_ids.indexOf(characterId)
  if (index >= 0) {
    referenceSelection.character_ids.splice(index, 1)
    return
  }
  referenceSelection.character_ids.push(characterId)
}

function clearCharacterReference() {
  referenceSelection.character_ids = []
}

function mergeIndexJobs(items: IndexJob[]) {
  if (items.length === 0) return
  const updatedIds = new Set(items.map((item) => item.id))
  workspace.indexJobs = [
    ...items,
    ...workspace.indexJobs.filter((item) => !updatedIds.has(item.id))
  ]
}

async function showContextPreviewNotice(target: PreviewTarget) {
  previewTarget.value = target
  previewResult.value = null
  previewError.value = t('editor.errors.previewContextRemoved')
  await nextTick()
  await focusPreviewResult()
}

async function requestChapterPlan() {
  localError.value = ''
  planStatus.value = 'idle'
  planning.value = true
  try {
    const ensuredChapterId = await ensureCurrentChapter()
    const agent = await resolveAgentForRole('plot-architect')
    const result = await api.runAgent(agent.id, {
      project_id: projectId.value,
      task_type: 'chapter_idea',
      input: {
        chapter_id: ensuredChapterId,
        brief: buildChapterIdeaBrief(),
        prompt: t('editor.planPromptPrefix'),
        title: title.value,
        style_constraints: styleConstraintList()
      },
      context_selection: selectionPayload.value,
      max_output_tokens: 1200
    })
    workspace.recordResult(t('editor.resultScopes.aiPlan'), result)
    agentRunResult.value = result.data
    chapterPlan.value = result.data.content.trim() || chapterPlan.value
    chapterIdeaWorkflowId.value = result.data.run.id
    activePanel.value = 'plan'
    planStatus.value = 'saved'
    await focusSidePanel()
    await refreshIndexJobs()
  } catch (error) {
    const apiError = workspace.recordError(t('editor.resultScopes.aiPlan'), error)
    localError.value = apiError.message || t('editor.errors.generatePlanFailed')
  } finally {
    planning.value = false
  }
}

async function requestDraft() {
  localError.value = ''
  draftStatus.value = 'idle'
  drafting.value = true
  try {
    const ensuredChapterId = await ensureCurrentChapter()
    const agent = await resolveAgentForRole('writer')
    const result = await api.runAgent(agent.id, {
      project_id: projectId.value,
      task_type: 'draft',
      input: {
        chapter_id: ensuredChapterId,
        brief: buildDraftBrief(),
        prompt: t('editor.draftPromptPrefix'),
        title: title.value,
        chapter_idea: chapterPlan.value,
        chapter_idea_run_id: chapterIdeaWorkflowId.value,
        style_constraints: styleConstraintList()
      },
      context_selection: selectionPayload.value
    })
    workspace.recordResult(t('editor.resultScopes.aiDraft'), result)
    agentRunResult.value = result.data
    content.value = [content.value.trim(), result.data.content.trim()].filter(Boolean).join('\n\n')
    activePanel.value = 'review'
    draftStatus.value = 'appended'
    await focusMainEditor()
    await refreshIndexJobs()
  } catch (error) {
    const apiError = workspace.recordError(t('editor.resultScopes.aiDraft'), error)
    localError.value = apiError.message || t('editor.errors.generateDraftFailed')
  } finally {
    drafting.value = false
  }
}

async function saveChapterVersion() {
  localError.value = ''
  saveStatus.value = 'idle'
  savingVersion.value = true
  try {
    const ensuredChapterId = await ensureCurrentChapter()
    const result = await api.saveChapterVersion(projectId.value, {
      chapter_id: ensuredChapterId,
      title: title.value,
      content: content.value,
      summary: content.value.slice(0, 180),
      author_role: 'editor',
      index_status: 'pending',
      metadata: { change_note: t('editor.changeNotes.manualSave'), chapter_plan: chapterPlan.value }
    })
    workspace.recordResult(t('editor.resultScopes.saveVersion'), result)
    versions.value = [result.data.chapter_version, ...versions.value.filter((item) => item.id !== result.data.chapter_version.id)]
    selectedVersionId.value = result.data.chapter_version.id
    loadedVersionId.value = result.data.chapter_version.id
    saveStatus.value = 'saved'
    mergeIndexJobs([result.data.index_job])
    await refreshIndexJobs()
  } catch (error) {
    const apiError = workspace.recordError(t('editor.resultScopes.saveVersion'), error)
    localError.value = apiError.message || t('editor.errors.saveVersionFailed')
  } finally {
    savingVersion.value = false
  }
}

function translatedStatusOrFallback(prefix: string, value: string) {
  const key = `${prefix}.${value}`
  const translated = t(key)
  return translated === key ? value : translated
}

function workflowStatusLabel(status: string) {
  return translatedStatusOrFallback('status.workflow', status)
}

function chapterStatusLabel(status: string) {
  return translatedStatusOrFallback('status.chapter', status)
}

function workflowStatusVariant(status: string) {
  if (status === 'succeeded' || status === 'completed') return 'success' as const
  if (status === 'failed') return 'rose' as const
  if (status === 'running') return 'gold' as const
  return 'muted' as const
}

function authorLabel(author: string) {
  return translatedStatusOrFallback('status.author', author)
}

function indexJobStatusLabel(status: string) {
  return translatedStatusOrFallback('status.indexJob', status)
}

function indexJobStatusVariant(status: string) {
  if (status === 'completed') return 'success' as const
  if (status === 'failed') return 'rose' as const
  if (status === 'running' || status === 'pending') return 'gold' as const
  return 'muted' as const
}

function freshnessStatusLabel(status: string) {
  return translatedStatusOrFallback('status.indexFreshness', status)
}

function freshnessStatusVariant(status: string) {
  if (status === 'fresh') return 'success' as const
  if (status === 'stale' || status === 'pending') return 'gold' as const
  return 'muted' as const
}

function modelResolutionSourceLabel(source: string) {
  return translatedStatusOrFallback('status.modelResolution', source)
}

function continuityAuditStatusLabel(status: string) {
  return translatedStatusOrFallback('editor.continuityAudit.status', status)
}

function continuityAuditStatusVariant(status: string) {
  if (status === 'passed') return 'success' as const
  if (status === 'warning') return 'gold' as const
  if (status === 'failed') return 'rose' as const
  return 'muted' as const
}

function continuityIssueTypeLabel(type: string) {
  return translatedStatusOrFallback('editor.continuityAudit.types', type)
}

function continuityIssueSeverityLabel(severity: string) {
  return translatedStatusOrFallback('editor.continuityAudit.severity', severity)
}

function continuityIssueSeverityVariant(severity: string) {
  if (severity === 'error') return 'rose' as const
  if (severity === 'warning') return 'gold' as const
  return 'muted' as const
}

function continuityEvidenceKindLabel(kind: string) {
  return translatedStatusOrFallback('editor.continuityAudit.evidenceKinds', kind)
}

function continuityAuditSummaryLabel() {
  if (!draftContinuityAudit.value) return t('editor.continuityAudit.summary.empty')
  if (continuityAuditPassed.value) return t('editor.continuityAudit.summary.passed')
  return t('editor.continuityAudit.summary.hasIssues', { count: continuityAuditIssues.value.length })
}

function versionWordCount(version: ChapterVersion) {
  return version.metrics?.word_count || version.content.replace(/\s/g, '').length
}
</script>

<template>
  <div class="min-w-0 space-y-6">
    <SectionHeader
      :title="t('editor.title')"
      :description="t('editor.description')"
    >
      <template #actions>
        <UiButton variant="outline" :to="`/projects/${projectId}`">{{ t('actions.back') }}</UiButton>
        <UiButton variant="outline" :disabled="planning" @click="requestChapterPlan">
          <Loader2 v-if="planning" class="h-4 w-4 animate-spin" />
          <Lightbulb v-else class="h-4 w-4" />
          {{ t('actions.generatePlan') }}
        </UiButton>
        <UiButton :disabled="drafting" @click="requestDraft">
          <Loader2 v-if="drafting" class="h-4 w-4 animate-spin" />
          <Sparkles v-else class="h-4 w-4" />
          {{ t('actions.continueDraft') }}
        </UiButton>
        <UiButton variant="archive" :disabled="savingVersion" @click="saveChapterVersion">
          <Loader2 v-if="savingVersion" class="h-4 w-4 animate-spin" />
          <Save v-else class="h-4 w-4" />
          {{ t('actions.saveVersion') }}
        </UiButton>
      </template>
    </SectionHeader>

    <StatusAlert :errors="workspace.errors" />
    <div v-if="localError" class="rounded-xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
      {{ localError }}
    </div>
    <div class="flex flex-wrap gap-2">
      <UiBadge v-if="saveStatus === 'saved'" variant="success">{{ t('editor.feedback.versionSaved') }}</UiBadge>
      <UiBadge v-if="draftStatus === 'appended'" variant="success">{{ t('editor.feedback.draftAppended') }}</UiBadge>
      <UiBadge v-if="planStatus === 'saved'" variant="success">{{ t('editor.feedback.planReady') }}</UiBadge>
    </div>

    <div class="grid min-w-0 gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(360px,460px)] 2xl:grid-cols-[minmax(0,1fr)_minmax(420px,520px)]">
      <UiCard ref="editorMainRef" tabindex="-1" class="min-w-0 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background">
        <div class="rounded-t-2xl border-b border-border bg-muted/35 p-4 sm:p-5">
          <div class="flex min-w-0 flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div class="min-w-0 flex-1 space-y-3">
              <div class="grid gap-2 md:grid-cols-[minmax(0,260px)_minmax(0,1fr)] md:items-end">
                <label class="space-y-1">
                  <span class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('editor.chapterSelector.label') }}</span>
                  <UiSelect
                    :model-value="chapterId"
                    :options="chapterOptions"
                    :placeholder="t('editor.chapterSelector.placeholder')"
                    searchable
                    :empty-text="t('editor.chapterSelector.empty')"
                    @update:model-value="selectChapter"
                  />
                </label>
                <p class="hidden truncate font-mono text-xs uppercase tracking-[0.18em] text-muted-foreground md:block" :title="chapterMetaLabel">{{ chapterMetaLabel }}</p>
              </div>
              <div v-if="hasInvalidRouteChapter" class="rounded-xl border border-amber-300/40 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-300/20 dark:bg-amber-300/10 dark:text-amber-100">
                {{ t('editor.invalidChapterNotice', { id: routeChapterId }) }}
              </div>
              <UiInput v-model="title" class="h-auto min-h-0 px-3 py-2 text-xl font-semibold leading-tight sm:text-2xl" />
            </div>
            <div class="flex min-w-0 flex-wrap gap-2">
              <UiBadge variant="muted">{{ t('editor.metrics.words', { count: metrics.wordCount }) }}</UiBadge>
              <UiBadge variant="muted">{{ t('editor.metrics.paragraphs', { count: metrics.paragraphs }) }}</UiBadge>
            </div>
          </div>
        </div>

        <div class="grid min-h-[520px] min-w-0 rounded-b-2xl lg:min-h-[700px] lg:grid-cols-[220px_minmax(0,1fr)]">
          <aside class="min-w-0 border-b border-border bg-muted/30 p-4 lg:border-b-0 lg:border-r">
            <p class="text-xs font-medium uppercase tracking-wide text-muted-foreground">{{ t('editor.outline.title') }}</p>
            <div class="mt-4 grid gap-3 sm:grid-cols-3 lg:block lg:space-y-3">
              <div v-for="card in outlineCards" :key="card.title" class="min-w-0 rounded-2xl border border-border bg-card p-3">
                <p class="break-words text-sm font-medium">{{ card.title }}</p>
                <p class="mt-1 break-words text-xs leading-5 text-muted-foreground">{{ card.description }}</p>
              </div>
            </div>
          </aside>

          <div class="min-w-0 space-y-5 p-4 sm:p-5">
            <div class="rounded-2xl border border-border bg-muted/20 p-4">
              <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <div>
                  <p class="text-sm font-medium">{{ t('editor.workflowHintTitle') }}</p>
                  <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ t('editor.workflowHintDescription') }}</p>
                </div>
                <UiBadge :variant="backgroundIndexState.variant">
                  {{ backgroundIndexState.label }}
                </UiBadge>
              </div>
              <div class="mt-4 flex flex-wrap gap-2">
                <UiBadge v-if="referenceFocusChips.length === 0" variant="muted">{{ t('editor.referenceFocusDefault') }}</UiBadge>
                <UiBadge v-for="chip in referenceFocusChips" :key="chip" variant="violet">{{ chip }}</UiBadge>
              </div>
            </div>
            <label class="block space-y-3">
              <span class="text-sm font-medium text-muted-foreground">{{ t('editor.chapterPlan') }}</span>
              <UiTextarea v-model="chapterPlan" :rows="9" class="min-h-44 text-sm leading-7 sm:min-h-52" />
            </label>
            <label class="block space-y-3">
              <span class="text-sm font-medium text-muted-foreground">{{ t('editor.content') }}</span>
              <UiTextarea v-model="content" :rows="22" class="min-h-[360px] font-serif text-base leading-8 sm:min-h-[440px]" />
            </label>
          </div>
        </div>
      </UiCard>

      <aside class="min-w-0 space-y-6">
        <UiCard ref="sidePanelRef" tabindex="-1" class="p-4 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background sm:p-5">
          <UiTabs v-model="activePanel" :tabs="tabs" class="w-full justify-start xl:justify-center" />

          <div v-if="activePanel === 'reference'" class="mt-5 min-w-0 space-y-4">
            <div>
              <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('editor.referenceFocusEyebrow') }}</p>
              <h2 class="mt-2 font-semibold">{{ t('editor.referenceFocusTitle') }}</h2>
              <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('editor.referenceFocusDescription') }}</p>
            </div>

            <div class="rounded-2xl border border-border bg-muted/25 p-4">
              <div class="grid gap-3">
                <UiSwitch
                  v-model="referenceSelection.include_chapter_plan"
                  :label="t('editor.referenceFocusOptions.chapterPlan.label')"
                  :description="t('editor.referenceFocusOptions.chapterPlan.description')"
                />
                <UiSwitch
                  v-model="referenceSelection.include_chapter_summary"
                  :label="t('editor.referenceFocusOptions.chapterSummary.label')"
                  :description="t('editor.referenceFocusOptions.chapterSummary.description')"
                />
                <UiSwitch
                  v-model="referenceSelection.include_world_rules"
                  :label="t('editor.referenceFocusOptions.worldRules.label')"
                  :description="t('editor.referenceFocusOptions.worldRules.description')"
                />
              </div>

              <div class="mt-4 rounded-2xl border border-border bg-card/80 p-4">
                <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <p class="text-sm font-medium text-foreground">{{ t('editor.referenceFocusOptions.previousChapters.label') }}</p>
                    <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ t('editor.referenceFocusOptions.previousChapters.description') }}</p>
                  </div>
                  <UiInput
                    v-model.number="referenceSelection.previous_chapter_count"
                    type="number"
                    min="0"
                    :max="previousChapterOptions.length"
                    class="w-full sm:w-28"
                  />
                </div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <UiBadge v-if="selectedPreviousChapters.length === 0" variant="muted">{{ t('editor.previousChapters.empty') }}</UiBadge>
                  <UiBadge v-for="chapter in selectedPreviousChapters" :key="chapter.id" variant="violet">
                    {{ chapter.title || chapter.id }}
                  </UiBadge>
                </div>
              </div>

              <div class="mt-4 rounded-2xl border border-border bg-card/80 p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p class="text-sm font-medium text-foreground">{{ t('editor.referenceFocusOptions.characters.label') }}</p>
                    <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ t('editor.referenceFocusOptions.characters.description') }}</p>
                  </div>
                  <UiButton
                    v-if="referenceSelection.character_ids.length"
                    type="button"
                    variant="outline"
                    class="h-8 px-3 text-xs"
                    @click="clearCharacterReference"
                  >
                    {{ t('editor.referenceFocusClear') }}
                  </UiButton>
                </div>

                <div v-if="availableCharacters.length === 0" class="mt-4 rounded-xl border border-border bg-muted/35 px-3 py-2 text-sm text-muted-foreground">
                  {{ t('editor.referenceFocusEmptyCharacters') }}
                </div>
                <div v-else class="mt-4 flex flex-wrap gap-2">
                  <button
                    v-for="character in availableCharacters"
                    :key="character.id"
                    type="button"
                    :class="cn(
                      'min-w-0 rounded-full border px-3 py-2 text-left text-sm transition-all focus-ring',
                      selectedCharacterIdSet.has(character.id)
                        ? 'border-primary/35 bg-primary/10 text-foreground'
                        : 'border-border bg-muted/35 text-muted-foreground hover:border-primary/30 hover:text-foreground'
                    )"
                    @click="toggleCharacterReference(character.id)"
                  >
                    <span class="block truncate font-medium">{{ character.name }}</span>
                    <span v-if="character.role" class="mt-1 block truncate text-[11px] text-muted-foreground">{{ character.role }}</span>
                    <span v-if="!character.entity_id" class="mt-1 block truncate text-[11px] text-amber-700 dark:text-amber-200">{{ t('editor.unsyncedCharacters.badge') }}</span>
                  </button>
                </div>

                <div class="mt-4 rounded-xl border border-border bg-muted/25 p-3">
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <p class="text-sm font-medium text-foreground">{{ t('editor.autoCharacters.title') }}</p>
                    <UiBadge :variant="hasManualCharacterSelection ? 'muted' : 'violet'">
                      {{ hasManualCharacterSelection ? t('editor.autoCharacters.manualOverride') : t('editor.autoCharacters.active') }}
                    </UiBadge>
                  </div>
                  <p class="mt-2 text-xs leading-5 text-muted-foreground">
                    {{ t('editor.autoCharacters.description') }}
                  </p>
                  <div v-if="autoSelectedCharacters.length === 0" class="mt-3 text-sm text-muted-foreground">
                    {{ t('editor.autoCharacters.empty') }}
                  </div>
                  <div v-else class="mt-3 flex flex-wrap gap-2">
                    <UiBadge v-for="character in autoSelectedCharacters" :key="character.id" variant="gold">
                      {{ character.name }}{{ character.entity_id ? '' : ` · ${t('editor.unsyncedCharacters.badge')}` }}
                    </UiBadge>
                  </div>
                </div>
              </div>
            </div>

            <div class="flex flex-wrap gap-2">
              <UiBadge v-if="referenceFocusChips.length === 0" variant="muted">{{ t('editor.referenceFocusDefault') }}</UiBadge>
              <UiBadge v-for="chip in referenceFocusChips" :key="chip" variant="violet">{{ chip }}</UiBadge>
            </div>

            <div v-if="hasUnsyncedRequestCharacters" class="rounded-2xl border border-amber-300/40 bg-amber-50 px-4 py-3 text-sm leading-6 text-amber-900 dark:border-amber-300/20 dark:bg-amber-300/10 dark:text-amber-100">
              {{ t('editor.unsyncedCharacters.message', { names: unsyncedRequestCharacterNames.join(t('common.listSeparator')) }) }}
            </div>

            <div v-if="referenceSections.length === 0" class="rounded-2xl border border-border bg-muted/35 p-4 text-sm text-muted-foreground">
              {{ t('editor.emptyReference') }}
            </div>
            <div v-else class="space-y-4">
              <div v-for="section in referenceSections" :key="section.key" class="rounded-2xl border border-border bg-muted/35 p-4">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ section.title }}</p>
                  <UiBadge v-if="section.emphasized" variant="violet">{{ t('editor.referenceFocusSelected') }}</UiBadge>
                </div>
                <ul class="mt-3 space-y-2 text-sm leading-6">
                  <li v-for="item in section.items" :key="item" class="break-words whitespace-pre-line">{{ item }}</li>
                </ul>
              </div>
            </div>

            <div class="rounded-2xl border border-border bg-muted/25 p-4">
              <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                <div>
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('editor.preview.eyebrow') }}</p>
                  <h3 class="mt-2 font-semibold">{{ t('editor.preview.title') }}</h3>
                  <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('editor.preview.description') }}</p>
                </div>
                <div class="flex flex-wrap gap-2">
                  <UiButton size="sm" variant="outline" :disabled="previewLoading" @click="showContextPreviewNotice('chapter_idea')">
                    <Loader2 v-if="previewLoading && previewTarget === 'chapter_idea'" class="h-4 w-4 animate-spin" />
                    <Lightbulb v-else class="h-4 w-4" />
                    {{ t('editor.preview.actions.chapter_idea') }}
                  </UiButton>
                  <UiButton size="sm" variant="outline" :disabled="previewLoading" @click="showContextPreviewNotice('draft')">
                    <Loader2 v-if="previewLoading && previewTarget === 'draft'" class="h-4 w-4 animate-spin" />
                    <Sparkles v-else class="h-4 w-4" />
                    {{ t('editor.preview.actions.draft') }}
                  </UiButton>
                </div>
              </div>

              <div class="mt-4 space-y-4">
                <div v-if="previewError" class="rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-3 text-sm text-destructive">
                  {{ previewError }}
                </div>
                <div v-else-if="previewLoading" class="rounded-xl border border-border bg-card/80 px-3 py-3 text-sm text-muted-foreground">
                  {{ t('editor.preview.loading') }}
                </div>
                <div v-else-if="!previewResult" class="rounded-xl border border-border bg-card/80 px-3 py-3 text-sm text-muted-foreground">
                  {{ t('editor.preview.empty') }}
                </div>
                <div v-else ref="previewResultRef" tabindex="-1" class="space-y-4 rounded-2xl outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background">
                  <div class="rounded-xl border border-border bg-card/80 p-4">
                    <div class="flex flex-wrap items-center justify-between gap-2">
                      <p class="text-sm font-medium text-foreground">{{ t('editor.preview.actualContext') }}</p>
                      <UiBadge variant="violet">{{ previewTargetLabelValue }}</UiBadge>
                    </div>
                    <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ previewSummary?.text || previewResult.summary }}</p>
                    <div class="mt-3 flex flex-wrap gap-2">
                      <UiBadge variant="muted">{{ t('editor.preview.counts.chapterSummaries', { count: previewSummary?.chapter_summary_count || 0 }) }}</UiBadge>
                      <UiBadge variant="muted">{{ t('editor.preview.counts.entities', { count: previewSummary?.entity_count || 0 }) }}</UiBadge>
                      <UiBadge variant="muted">{{ t('editor.preview.counts.facts', { count: previewSummary?.fact_count || 0 }) }}</UiBadge>
                      <UiBadge variant="muted">{{ t('editor.preview.counts.plotThreads', { count: previewSummary?.plot_thread_count || 0 }) }}</UiBadge>
                      <UiBadge variant="muted">{{ t('editor.preview.counts.worldRules', { count: previewSummary?.world_rule_count || 0 }) }}</UiBadge>
                      <UiBadge variant="muted">{{ t('editor.preview.estimatedTokens', { count: previewResult.estimated_tokens }) }}</UiBadge>
                    </div>
                  </div>

                  <div class="grid gap-4 md:grid-cols-2">
                    <div class="rounded-xl border border-border bg-card/80 p-4">
                      <div class="flex flex-wrap items-center justify-between gap-2">
                        <p class="text-sm font-medium text-foreground">{{ t('editor.preview.freshnessTitle') }}</p>
                        <UiBadge :variant="freshnessStatusVariant(previewResult.index_freshness.status)">
                          {{ freshnessStatusLabel(previewResult.index_freshness.status) }}
                        </UiBadge>
                      </div>
                      <dl class="mt-3 space-y-2 text-sm leading-6">
                        <div class="flex items-start justify-between gap-3">
                          <dt class="text-muted-foreground">{{ t('editor.freshness.pendingJobs') }}</dt>
                          <dd class="text-right">{{ previewResult.index_freshness.pending_job_count }}</dd>
                        </div>
                        <div class="flex items-start justify-between gap-3">
                          <dt class="text-muted-foreground">{{ t('editor.freshness.latestChapterVersion') }}</dt>
                          <dd class="text-right break-all">{{ previewResult.index_freshness.latest_chapter_version_id || t('common.emptyValue') }}</dd>
                        </div>
                        <div class="flex items-start justify-between gap-3">
                          <dt class="text-muted-foreground">{{ t('editor.freshness.latestIndexedVersion') }}</dt>
                          <dd class="text-right break-all">{{ previewResult.index_freshness.latest_indexed_chapter_version_id || t('common.emptyValue') }}</dd>
                        </div>
                      </dl>
                    </div>

                    <div class="rounded-xl border border-border bg-card/80 p-4">
                      <div class="flex flex-wrap items-center justify-between gap-2">
                        <p class="text-sm font-medium text-foreground">{{ t('editor.preview.modelResolutionTitle') }}</p>
                        <UiBadge variant="muted">{{ modelResolutionSourceLabel(previewResult.model_resolution.resolution_source) }}</UiBadge>
                      </div>
                      <dl class="mt-3 space-y-2 text-sm leading-6">
                        <div class="flex items-start justify-between gap-3">
                          <dt class="text-muted-foreground">{{ t('editor.diagnostics.provider') }}</dt>
                          <dd class="text-right break-all">{{ previewResult.model_resolution.provider_name || previewResult.model_resolution.provider_id || t('common.emptyValue') }}</dd>
                        </div>
                        <div class="flex items-start justify-between gap-3">
                          <dt class="text-muted-foreground">{{ t('editor.diagnostics.model') }}</dt>
                          <dd class="text-right break-all">{{ previewResult.model_resolution.model_name || previewResult.model_resolution.model_id || t('common.emptyValue') }}</dd>
                        </div>
                        <div class="flex items-start justify-between gap-3">
                          <dt class="text-muted-foreground">{{ t('editor.diagnostics.route') }}</dt>
                          <dd class="text-right break-all">{{ previewResult.model_resolution.route_key || t('common.emptyValue') }}</dd>
                        </div>
                      </dl>
                    </div>
                  </div>

                  <div class="rounded-xl border border-border bg-card/80 p-4">
                    <div class="flex flex-wrap items-center justify-between gap-2">
                      <p class="text-sm font-medium text-foreground">{{ t('editor.toolTrace.title') }}</p>
                      <UiBadge variant="muted">{{ previewToolTraceSummary.text }}</UiBadge>
                    </div>
                    <div v-if="!previewToolTraceSummary.hasTrace" class="mt-3 text-sm text-muted-foreground">
                      {{ t('editor.toolTrace.empty') }}
                    </div>
                    <div v-else class="mt-3 space-y-2">
                      <div v-for="trace in previewToolTraceSummary.items" :key="trace.id" class="rounded-xl border border-border bg-muted/20 p-3">
                        <div class="flex flex-wrap items-center gap-2">
                          <UiBadge variant="violet">{{ toolTraceCategoryLabel(trace.category) }}</UiBadge>
                          <span class="min-w-0 break-words text-sm font-medium text-foreground">{{ trace.name }}</span>
                        </div>
                        <p class="mt-2 break-words text-xs leading-5 text-muted-foreground">{{ trace.summary }}</p>
                      </div>
                    </div>
                  </div>

                  <div class="rounded-xl border border-border bg-card/80 p-4">
                    <div class="flex flex-wrap items-center justify-between gap-2">
                      <p class="text-sm font-medium text-foreground">{{ t('editor.preview.structuredContextTitle') }}</p>
                      <UiBadge v-if="previewStructuredContext" variant="muted">{{ t('editor.preview.actualContext') }}</UiBadge>
                    </div>
                    <div v-if="!previewStructuredContext" class="mt-3 text-sm text-muted-foreground">
                      {{ t('editor.preview.emptyStructuredContext') }}
                    </div>
                    <div v-else class="mt-3 space-y-3">
                      <div v-for="section in previewContextSections" :key="section.key" class="rounded-xl border border-border bg-muted/20 p-3">
                        <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ section.title }}</p>
                        <ul class="mt-2 max-h-40 space-y-2 overflow-auto text-sm leading-6 subtle-scrollbar">
                          <li v-for="item in section.items" :key="item" class="break-words whitespace-pre-line">{{ item }}</li>
                        </ul>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="activePanel === 'plan'" class="mt-5 min-w-0 space-y-4">
            <div class="flex items-center gap-2">
              <Lightbulb class="h-5 w-5 text-muted-foreground" />
              <h2 class="font-semibold">{{ t('editor.plan') }}</h2>
            </div>
            <UiBadge v-if="planStatus === 'saved'" variant="success">{{ t('editor.feedback.planReady') }}</UiBadge>
            <p class="text-sm leading-6 text-muted-foreground">{{ t('editor.planDescription') }}</p>
            <label class="space-y-2">
              <span class="text-sm text-muted-foreground">{{ t('editor.instruction') }}</span>
              <UiTextarea v-model="prompt" :rows="5" />
            </label>
            <label class="space-y-2">
              <span class="text-sm text-muted-foreground">{{ t('editor.chapterPlan') }}</span>
              <UiTextarea v-model="chapterPlan" :rows="8" />
            </label>
            <label class="space-y-2">
              <span class="text-sm text-muted-foreground">{{ t('editor.style') }}</span>
              <UiInput v-model="styleConstraints" />
            </label>
            <UiButton class="w-full" variant="outline" :disabled="planning" @click="requestChapterPlan">
              <Loader2 v-if="planning" class="h-4 w-4 animate-spin" />
              <Lightbulb v-else class="h-4 w-4" />
              {{ t('actions.generatePlan') }}
            </UiButton>
          </div>

          <div v-else-if="activePanel === 'draft'" class="mt-5 min-w-0 space-y-4">
            <div class="flex items-center gap-2">
              <Bot class="h-5 w-5 text-muted-foreground" />
              <h2 class="font-semibold">{{ t('editor.draft') }}</h2>
            </div>
            <UiBadge v-if="draftStatus === 'appended'" variant="success">{{ t('editor.feedback.draftAppended') }}</UiBadge>
            <p class="text-sm leading-6 text-muted-foreground">{{ t('editor.draftDescription') }}</p>
            <div class="rounded-2xl border border-border bg-muted/35 p-4 text-sm leading-6 text-muted-foreground">
              <p class="font-medium text-foreground">{{ t('editor.currentPlanPreview') }}</p>
              <p class="mt-2 max-h-64 overflow-auto whitespace-pre-line break-words subtle-scrollbar">{{ chapterPlan || t('common.emptyValue') }}</p>
            </div>
            <UiButton class="w-full" :disabled="drafting" @click="requestDraft">
              <Loader2 v-if="drafting" class="h-4 w-4 animate-spin" />
              <Sparkles v-else class="h-4 w-4" />
              {{ t('actions.continueDraft') }}
            </UiButton>
          </div>

          <div v-else-if="activePanel === 'review'" class="mt-5 min-w-0 space-y-4">
            <div class="flex items-center gap-2">
              <BrainCircuit class="h-5 w-5 text-muted-foreground" />
              <h2 class="font-semibold">{{ t('editor.review') }}</h2>
            </div>

            <div class="rounded-2xl border border-border bg-muted/35 p-4">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('editor.continuityAudit.title') }}</p>
                  <h3 class="mt-2 font-semibold">{{ t('editor.continuityAudit.title') }}</h3>
                  <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('editor.continuityAudit.description') }}</p>
                </div>
                <div class="flex flex-wrap items-center gap-2">
                  <UiBadge
                    v-if="draftContinuityAudit"
                    :variant="continuityAuditStatusVariant(draftContinuityAudit.status)"
                  >
                    {{ continuityAuditStatusLabel(draftContinuityAudit.status) }}
                  </UiBadge>
                  <UiBadge v-if="draftContinuityAudit && !continuityAuditPassed" variant="muted">
                    {{ t('editor.continuityAudit.issueCount', { count: continuityAuditIssues.length }) }}
                  </UiBadge>
                </div>
              </div>

              <div class="mt-4 rounded-xl border border-border bg-card/80 px-3 py-3 text-sm text-muted-foreground">
                {{ continuityAuditSummaryLabel() }}
              </div>

              <div v-if="draftContinuityAudit && continuityAuditPassed" class="mt-4 rounded-xl border border-emerald-300/40 bg-emerald-50 px-3 py-3 text-sm text-emerald-900 dark:border-emerald-300/20 dark:bg-emerald-300/10 dark:text-emerald-100">
                {{ t('editor.continuityAudit.passed') }}
              </div>

              <div v-if="draftContinuityAudit && continuityAuditIssues.length > 0" class="mt-4 space-y-3">
                <div v-for="(issue, issueIndex) in continuityAuditIssues" :key="`${issue.type}-${issueIndex}`" class="rounded-xl border border-border bg-card/80 p-4">
                  <div class="flex flex-wrap items-start justify-between gap-2">
                    <div class="min-w-0 flex-1">
                      <p class="break-words text-sm font-medium text-foreground">{{ continuityIssueTypeLabel(issue.type) }}</p>
                      <p class="mt-1 text-xs text-muted-foreground">{{ issue.type }}</p>
                    </div>
                    <UiBadge class="shrink-0" :variant="continuityIssueSeverityVariant(issue.severity)">
                      {{ continuityIssueSeverityLabel(issue.severity) }}
                    </UiBadge>
                  </div>

                  <dl class="mt-4 space-y-3 text-sm leading-6">
                    <div>
                      <dt class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('editor.continuityAudit.message') }}</dt>
                      <dd class="mt-1 break-words text-foreground">{{ issue.message }}</dd>
                    </div>
                    <div>
                      <dt class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('editor.continuityAudit.excerpt') }}</dt>
                      <dd class="mt-1 rounded-lg border border-border/70 bg-muted/20 px-3 py-2 break-words whitespace-pre-line text-foreground">{{ issue.draft_excerpt || t('common.emptyValue') }}</dd>
                    </div>
                    <div>
                      <dt class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('editor.continuityAudit.suggestion') }}</dt>
                      <dd class="mt-1 break-words whitespace-pre-line text-foreground">{{ issue.suggestion || t('common.emptyValue') }}</dd>
                    </div>
                    <div>
                      <dt class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('editor.continuityAudit.evidence') }}</dt>
                      <dd class="mt-2">
                        <div v-if="issue.evidence.length === 0" class="text-muted-foreground">{{ t('common.emptyValue') }}</div>
                        <div v-else class="space-y-2">
                          <div v-for="(evidence, evidenceIndex) in issue.evidence" :key="`${evidence.source_type}-${evidence.source_id || evidence.label}-${evidenceIndex}`" class="rounded-xl border border-border bg-muted/20 p-3">
                            <div class="flex flex-wrap items-center gap-2">
                              <UiBadge variant="muted">{{ continuityEvidenceKindLabel(evidence.source_type) }}</UiBadge>
                              <span class="min-w-0 break-words text-sm font-medium text-foreground">{{ evidence.label }}</span>
                            </div>
                            <p v-if="evidence.excerpt" class="mt-2 rounded-lg border border-border/70 bg-card/70 px-3 py-2 break-words whitespace-pre-line text-sm text-muted-foreground">{{ evidence.excerpt }}</p>
                          </div>
                        </div>
                      </dd>
                    </div>
                  </dl>
                </div>
              </div>
            </div>

            <div class="rounded-2xl border border-border bg-muted/35 p-4">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('editor.diagnostics.eyebrow') }}</p>
                  <h3 class="mt-2 font-semibold">{{ t('editor.diagnostics.title') }}</h3>
                </div>
                <UiBadge variant="muted">{{ diagnosticsExecutionTarget }}</UiBadge>
              </div>
              <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('editor.diagnostics.description') }}</p>

              <div v-if="!diagnosticsModelResolution && !diagnosticsFreshness && latestIndexJobs.length === 0" class="mt-4 rounded-xl border border-border bg-card/80 px-3 py-3 text-sm text-muted-foreground">
                {{ t('editor.diagnostics.empty') }}
              </div>

              <div v-else class="mt-4 space-y-4">
                <div class="grid gap-4 md:grid-cols-2">
                  <div class="rounded-xl border border-border bg-card/80 p-4">
                    <div class="flex flex-wrap items-center justify-between gap-2">
                      <p class="text-sm font-medium text-foreground">{{ t('editor.diagnostics.modelResolution') }}</p>
                      <UiBadge v-if="diagnosticsModelResolution" variant="muted">{{ modelResolutionSourceLabel(diagnosticsModelResolution.resolution_source) }}</UiBadge>
                    </div>
                    <div v-if="!diagnosticsModelResolution" class="mt-3 text-sm text-muted-foreground">
                      {{ t('editor.diagnostics.emptyModelResolution') }}
                    </div>
                    <dl v-else class="mt-3 space-y-2 text-sm leading-6">
                      <div class="flex items-start justify-between gap-3">
                        <dt class="text-muted-foreground">{{ t('editor.diagnostics.provider') }}</dt>
                        <dd class="text-right break-all">{{ diagnosticsModelResolution.provider_name || diagnosticsModelResolution.provider_id || t('common.emptyValue') }}</dd>
                      </div>
                      <div class="flex items-start justify-between gap-3">
                        <dt class="text-muted-foreground">{{ t('editor.diagnostics.model') }}</dt>
                        <dd class="text-right break-all">{{ diagnosticsModelResolution.model_name || diagnosticsModelResolution.model_id || t('common.emptyValue') }}</dd>
                      </div>
                      <div class="flex items-start justify-between gap-3">
                        <dt class="text-muted-foreground">{{ t('editor.diagnostics.route') }}</dt>
                        <dd class="text-right break-all">{{ diagnosticsModelResolution.route_key || t('common.emptyValue') }}</dd>
                      </div>
                      <div class="flex items-start justify-between gap-3">
                        <dt class="text-muted-foreground">{{ t('editor.diagnostics.routeSource') }}</dt>
                        <dd class="text-right">{{ modelResolutionSourceLabel(diagnosticsModelResolution.resolution_source) }}</dd>
                      </div>
                    </dl>
                  </div>

                  <div class="rounded-xl border border-border bg-card/80 p-4">
                    <div class="flex flex-wrap items-center justify-between gap-2">
                      <p class="text-sm font-medium text-foreground">{{ t('editor.diagnostics.freshness') }}</p>
                      <UiBadge v-if="diagnosticsFreshness" :variant="freshnessStatusVariant(diagnosticsFreshness.status)">
                        {{ freshnessStatusLabel(diagnosticsFreshness.status) }}
                      </UiBadge>
                    </div>
                    <div v-if="!diagnosticsFreshness" class="mt-3 text-sm text-muted-foreground">
                      {{ t('editor.diagnostics.emptyFreshness') }}
                    </div>
                    <dl v-else class="mt-3 space-y-2 text-sm leading-6">
                      <div class="flex items-start justify-between gap-3">
                        <dt class="text-muted-foreground">{{ t('editor.freshness.pendingJobs') }}</dt>
                        <dd class="text-right">{{ diagnosticsFreshness.pending_job_count }}</dd>
                      </div>
                      <div class="flex items-start justify-between gap-3">
                        <dt class="text-muted-foreground">{{ t('editor.freshness.latestChapterVersion') }}</dt>
                        <dd class="text-right break-all">{{ diagnosticsFreshness.latest_chapter_version_id || t('common.emptyValue') }}</dd>
                      </div>
                      <div class="flex items-start justify-between gap-3">
                        <dt class="text-muted-foreground">{{ t('editor.freshness.latestIndexedVersion') }}</dt>
                        <dd class="text-right break-all">{{ diagnosticsFreshness.latest_indexed_chapter_version_id || t('common.emptyValue') }}</dd>
                      </div>
                    </dl>
                  </div>
                </div>

                <div class="rounded-xl border border-border bg-card/80 p-4">
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <p class="text-sm font-medium text-foreground">{{ t('editor.toolTrace.title') }}</p>
                    <UiBadge variant="muted">{{ toolTraceSummary.text }}</UiBadge>
                  </div>
                  <div v-if="!toolTraceSummary.hasTrace" class="mt-3 text-sm text-muted-foreground">
                    {{ t('editor.toolTrace.empty') }}
                  </div>
                  <div v-else class="mt-3 grid gap-3 md:grid-cols-2">
                    <div v-for="trace in toolTraceSummary.items" :key="trace.id" class="rounded-xl border border-border bg-muted/20 p-3">
                      <div class="flex flex-wrap items-center gap-2">
                        <UiBadge variant="violet">{{ toolTraceCategoryLabel(trace.category) }}</UiBadge>
                        <span class="min-w-0 break-words text-sm font-medium text-foreground">{{ trace.name }}</span>
                      </div>
                      <p class="mt-2 break-words text-xs leading-5 text-muted-foreground">{{ trace.summary }}</p>
                    </div>
                  </div>
                </div>

                <div class="rounded-xl border border-border bg-card/80 p-4">
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <p class="text-sm font-medium text-foreground">{{ t('editor.diagnostics.latestIndexJobs') }}</p>
                    <UiBadge :variant="backgroundIndexState.variant">{{ backgroundIndexState.label }}</UiBadge>
                  </div>
                  <div v-if="latestIndexJobs.length === 0" class="mt-3 text-sm text-muted-foreground">
                    {{ t('editor.diagnostics.emptyIndexJobs') }}
                  </div>
                  <div v-else class="mt-3 space-y-3">
                    <div v-for="job in latestIndexJobs" :key="job.id" class="rounded-xl border border-border bg-muted/20 p-3">
                      <div class="flex flex-wrap items-center justify-between gap-3">
                        <p class="min-w-0 flex-1 break-words font-medium">{{ job.kind }}</p>
                        <UiBadge class="shrink-0" :variant="indexJobStatusVariant(job.status)">{{ indexJobStatusLabel(job.status) }}</UiBadge>
                      </div>
                      <p class="mt-2 break-words font-mono text-xs text-muted-foreground" :title="t('editor.jobAttempts', { id: job.id, count: job.attempts })">{{ t('editor.jobAttempts', { id: job.id, count: job.attempts }) }}</p>
                      <p v-if="job.error" class="mt-2 break-words rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">{{ job.error }}</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div v-for="stepItem in workflowSteps" :key="stepItem.id" class="min-w-0 rounded-2xl border border-border bg-muted/35 p-4">
              <div class="flex min-w-0 flex-wrap items-center justify-between gap-3">
                <p class="min-w-0 flex-1 break-words font-medium">{{ stepItem.name }}</p>
                <UiBadge class="shrink-0" :variant="workflowStatusVariant(stepItem.status)">
                  {{ workflowStatusLabel(stepItem.status) }}
                </UiBadge>
              </div>
              <p class="mt-2 break-words text-sm leading-6 text-muted-foreground">{{ stepItem.message }}</p>
            </div>

            <div v-if="draftWarnings.length" class="space-y-2">
              <p v-for="warning in draftWarnings" :key="warning" class="rounded-xl border border-amber-300/40 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-300/20 dark:bg-amber-300/10 dark:text-amber-100">{{ warning }}</p>
            </div>
          </div>

          <div v-else-if="activePanel === 'versions'" class="mt-5 min-w-0 space-y-4">
            <div class="flex min-w-0 flex-wrap items-center justify-between gap-3">
              <div class="min-w-0 flex-1">
                <p class="field-label text-xs uppercase tracking-[0.18em]">
                  {{ t('editor.versionsEyebrow') }}
                  <UiInfoTooltip :text="t('tooltips.editorVersions')" />
                </p>
                <h2 class="mt-2 break-words text-lg font-semibold">{{ t('editor.versions') }}</h2>
              </div>
              <FileClock :class="['h-5 w-5 shrink-0 text-muted-foreground', loadingVersions && 'animate-pulse']" />
            </div>
            <div v-if="versions.length === 0" class="rounded-2xl border border-border bg-muted/35 p-4 text-sm text-muted-foreground">
              {{ t('editor.emptyVersions') }}
            </div>
            <div v-else class="space-y-3">
              <button
                v-for="version in versions"
                :key="version.id"
                type="button"
                :class="[
                  'w-full min-w-0 rounded-2xl border p-4 text-left transition-all hover:border-primary/35 focus-ring',
                  loadedVersionId === version.id
                    ? 'border-primary/45 bg-primary/10 shadow-sm shadow-primary/10'
                    : 'border-border bg-muted/35'
                ]"
                @click="loadVersion(version)"
              >
                <div class="flex min-w-0 flex-wrap items-center justify-between gap-3">
                  <p class="min-w-0 flex-1 break-words font-medium" :title="t('editor.versionLabel', { version: version.version, title: version.title })">{{ t('editor.versionLabel', { version: version.version, title: version.title }) }}</p>
                  <div class="flex shrink-0 flex-wrap justify-end gap-2">
                    <UiBadge v-if="loadedVersionId === version.id" variant="violet">{{ t('editor.feedback.versionLoaded') }}</UiBadge>
                    <UiBadge :variant="version.author === 'ai' ? 'default' : 'muted'">{{ authorLabel(version.author) }}</UiBadge>
                  </div>
                </div>
                <p class="mt-2 break-words text-xs text-muted-foreground" :title="`${formatDateTime(version.created_at)} · ${version.change_note}`">{{ formatDateTime(version.created_at) }} · {{ version.change_note }}</p>
                <div class="mt-3 flex flex-wrap gap-2">
                  <UiBadge variant="muted">{{ t('editor.metrics.words', { count: versionWordCount(version) }) }}</UiBadge>
                  <UiBadge variant="muted">{{ version.index_status || t('common.emptyValue') }}</UiBadge>
                </div>
              </button>
            </div>
          </div>

          <div v-else class="mt-5 min-w-0 space-y-4">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p class="field-label text-xs uppercase tracking-[0.18em]">
                  {{ t('editor.runLog') }}
                  <UiInfoTooltip :text="t('tooltips.runLog')" />
                </p>
                <h2 class="mt-2 font-semibold">{{ t('editor.runLogTitle') }}</h2>
                <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('editor.runLogDescription') }}</p>
              </div>
              <UiBadge :variant="backgroundIndexState.variant">{{ backgroundIndexState.label }}</UiBadge>
            </div>

            <div class="grid gap-4">
              <div class="rounded-2xl border border-border bg-muted/35 p-4">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <p class="field-label text-sm font-medium text-foreground">
                    {{ t('editor.diagnostics.modelResolution') }}
                    <UiInfoTooltip :text="t('tooltips.modelResolution')" />
                  </p>
                  <UiBadge v-if="diagnosticsModelResolution" variant="muted">{{ modelResolutionSourceLabel(diagnosticsModelResolution.resolution_source) }}</UiBadge>
                </div>
                <div v-if="!diagnosticsModelResolution" class="mt-3 text-sm text-muted-foreground">
                  {{ t('editor.diagnostics.emptyModelResolution') }}
                </div>
                <dl v-else class="mt-3 space-y-2 text-sm leading-6">
                  <div class="flex items-start justify-between gap-3">
                    <dt class="field-label text-muted-foreground">{{ t('editor.diagnostics.provider') }}<UiInfoTooltip :text="t('tooltips.providerType')" /></dt>
                    <dd class="text-right break-all">{{ diagnosticsModelResolution.provider_name || diagnosticsModelResolution.provider_id || t('common.emptyValue') }}</dd>
                  </div>
                  <div class="flex items-start justify-between gap-3">
                    <dt class="field-label text-muted-foreground">{{ t('editor.diagnostics.model') }}<UiInfoTooltip :text="t('tooltips.modelKind')" /></dt>
                    <dd class="text-right break-all">{{ diagnosticsModelResolution.model_name || diagnosticsModelResolution.model_id || t('common.emptyValue') }}</dd>
                  </div>
                  <div class="flex items-start justify-between gap-3">
                    <dt class="field-label text-muted-foreground">{{ t('editor.diagnostics.route') }}<UiInfoTooltip :text="t('tooltips.modelRouting')" /></dt>
                    <dd class="text-right break-all">{{ diagnosticsModelResolution.route_key || t('common.emptyValue') }}</dd>
                  </div>
                </dl>
              </div>

              <div class="rounded-2xl border border-border bg-muted/35 p-4">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <p class="field-label text-sm font-medium text-foreground">
                    {{ t('editor.diagnostics.freshness') }}
                    <UiInfoTooltip :text="t('tooltips.indexFreshness')" />
                  </p>
                  <UiBadge v-if="diagnosticsFreshness" :variant="freshnessStatusVariant(diagnosticsFreshness.status)">
                    {{ freshnessStatusLabel(diagnosticsFreshness.status) }}
                  </UiBadge>
                </div>
                <div v-if="!diagnosticsFreshness" class="mt-3 text-sm text-muted-foreground">
                  {{ t('editor.diagnostics.emptyFreshness') }}
                </div>
                <dl v-else class="mt-3 space-y-2 text-sm leading-6">
                  <div class="flex items-start justify-between gap-3">
                    <dt class="text-muted-foreground">{{ t('editor.freshness.pendingJobs') }}</dt>
                    <dd class="text-right">{{ diagnosticsFreshness.pending_job_count }}</dd>
                  </div>
                  <div class="flex items-start justify-between gap-3">
                    <dt class="text-muted-foreground">{{ t('editor.freshness.latestChapterVersion') }}</dt>
                    <dd class="text-right break-all">{{ diagnosticsFreshness.latest_chapter_version_id || t('common.emptyValue') }}</dd>
                  </div>
                  <div class="flex items-start justify-between gap-3">
                    <dt class="text-muted-foreground">{{ t('editor.freshness.latestIndexedVersion') }}</dt>
                    <dd class="text-right break-all">{{ diagnosticsFreshness.latest_indexed_chapter_version_id || t('common.emptyValue') }}</dd>
                  </div>
                </dl>
              </div>

              <div class="rounded-2xl border border-border bg-muted/35 p-4">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <p class="field-label text-sm font-medium text-foreground">
                    {{ t('editor.toolTrace.title') }}
                    <UiInfoTooltip :text="t('tooltips.toolTrace')" />
                  </p>
                  <UiBadge variant="muted">{{ toolTraceSummary.text }}</UiBadge>
                </div>
                <div v-if="!toolTraceSummary.hasTrace" class="mt-3 text-sm text-muted-foreground">
                  {{ t('editor.toolTrace.empty') }}
                </div>
                <div v-else class="mt-3 space-y-3">
                  <div v-for="trace in toolTraceSummary.items" :key="trace.id" class="rounded-xl border border-border bg-card/80 p-3">
                    <div class="flex flex-wrap items-center gap-2">
                      <UiBadge variant="violet">{{ toolTraceCategoryLabel(trace.category) }}</UiBadge>
                      <span class="min-w-0 break-words text-sm font-medium text-foreground">{{ trace.name }}</span>
                    </div>
                    <p class="mt-2 break-words text-xs leading-5 text-muted-foreground">{{ trace.summary }}</p>
                  </div>
                </div>
              </div>

              <div class="rounded-2xl border border-border bg-muted/35 p-4">
                <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <p class="field-label text-sm font-medium text-foreground">
                    {{ t('editor.indexJobs') }}
                    <UiInfoTooltip :text="t('tooltips.indexJobs')" />
                  </p>
                  <UiButton size="sm" variant="outline" @click="refreshIndexJobs">
                    <RefreshCw class="h-4 w-4" />
                    {{ t('actions.refresh') }}
                  </UiButton>
                </div>
                <AppTaskBoard :jobs="projectIndexJobs" project-scoped />
              </div>

              <div class="space-y-3">
                <div v-for="stepItem in workflowSteps" :key="stepItem.id" class="min-w-0 rounded-2xl border border-border bg-muted/35 p-4">
                  <div class="flex min-w-0 flex-wrap items-center justify-between gap-3">
                    <p class="min-w-0 flex-1 break-words font-medium">{{ stepItem.name }}</p>
                    <UiBadge class="shrink-0" :variant="workflowStatusVariant(stepItem.status)">
                      {{ workflowStatusLabel(stepItem.status) }}
                    </UiBadge>
                  </div>
                  <p class="mt-2 break-words text-sm leading-6 text-muted-foreground">{{ stepItem.message }}</p>
                </div>
              </div>
            </div>
          </div>
        </UiCard>

      </aside>
    </div>
  </div>
</template>
