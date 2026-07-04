<script setup lang="ts">
import { ArrowRight, BookMarked, CheckCircle2, FilePenLine, FileText, GitFork, Loader2, PenLine, Plus, Save, ShieldCheck, Sparkles, Trash2, UserRound, WifiOff } from '@lucide/vue'
import { storeToRefs } from 'pinia'
import type { CharacterProfile, CharacterProfileMode, StoryBible } from '~/lib/types'

const route = useRoute()
const { t } = useI18n()
const projectId = computed(() => String(route.params.id))
const api = useApi()
const workspace = useWorkspaceStore()
const { activeBible, errors, loading } = storeToRefs(workspace)

const bibleDraft = ref<StoryBible | null>(null)
const bibleSaveState = ref<'idle' | 'saving' | 'saved' | 'failed'>('idle')
const characterSyncState = ref<'idle' | 'syncing' | 'synced' | 'failed'>('idle')
const characterGenerationState = ref<'idle' | 'generating' | 'failed'>('idle')
const openingChapterId = ref('')
const generatedCharacters = ref<CharacterProfile[]>([])
const activeSectionValues = ['overview', 'story', 'characters', 'chapters'] as const

type ActiveSection = typeof activeSectionValues[number]

const activeSection = ref<ActiveSection>('overview')

const foreshadowStatusValues: Array<StoryBible['foreshadows'][number]['status']> = ['planted', 'active', 'paid_off']
const chapterStatusValues: Array<StoryBible['chapters'][number]['status']> = ['planned', 'drafting', 'reviewing', 'locked']

onMounted(() => workspace.loadStoryBible(projectId.value))
watch(projectId, (id) => workspace.loadStoryBible(id))
watch(activeBible, (bible) => {
  bibleDraft.value = bible ? cloneBible(bible) : null
  bibleSaveState.value = 'idle'
}, { immediate: true })

const bible = computed(() => bibleDraft.value || activeBible.value)
const projectSummary = computed(() => {
  if (!bible.value) return []

  return [
    {
      label: t('projectOverview.summary.premise'),
      value: bible.value.premise || t('common.emptyValue')
    },
    {
      label: t('projectOverview.summary.themes'),
      value: bible.value.themes.length ? bible.value.themes.join(t('common.listSeparator')) : t('common.emptyValue')
    },
    {
      label: t('projectOverview.summary.worldRules'),
      value: bible.value.world_rules.length ? bible.value.world_rules.slice(0, 3).join(`\n`) : t('common.emptyValue')
    }
  ]
})

const chapterProgressSummary = computed(() => {
  if (!bible.value) return null

  const activeCount = bible.value.chapters.filter((chapter) => chapter.status !== 'planned').length
  const nextChapter = bible.value.chapters.find((chapter) => chapter.status === 'drafting') || bible.value.chapters[0]

  return {
    total: bible.value.chapters.length,
    active: activeCount,
    nextChapter
  }
})

const prepCards = computed(() => {
  if (!bible.value) return []

  return [
    {
      key: 'rules',
      title: t('projectOverview.prep.rules.title'),
      description: t('projectOverview.prep.rules.description'),
      count: bible.value.world_rules.length,
      action: () => setActiveSection('story')
    },
    {
      key: 'characters',
      title: t('projectOverview.prep.characters.title'),
      description: t('projectOverview.prep.characters.description'),
      count: bible.value.characters.length,
      action: () => setActiveSection('characters')
    },
    {
      key: 'foreshadows',
      title: t('projectOverview.prep.foreshadows.title'),
      description: t('projectOverview.prep.foreshadows.description'),
      count: bible.value.foreshadows.length,
      action: () => setActiveSection('story')
    }
  ]
})

const chapterCards = computed(() => {
  if (!bible.value) return []
  return bible.value.chapters
})

const sectionTabs = computed(() => [
  { label: t('projectOverview.sections.overview'), value: 'overview' },
  { label: t('projectOverview.sections.story'), value: 'story' },
  { label: t('projectOverview.sections.characters'), value: 'characters' },
  { label: t('projectOverview.sections.chapters'), value: 'chapters', badge: String(bible.value?.chapters.length || 0) }
])

const foreshadowStatusOptions = computed(() => foreshadowStatusValues.map((status) => ({ label: foreshadowStatusLabel(status), value: status })))
const chapterStatusOptions = computed(() => chapterStatusValues.map((status) => ({ label: chapterStatusLabel(status), value: status })))

function cloneBible(bible: StoryBible): StoryBible {
  return JSON.parse(JSON.stringify(bible)) as StoryBible
}

function createUniqueId(prefix: string, existingIds: string[]) {
  const existing = new Set(existingIds)
  let index = existing.size + 1
  while (existing.has(`${prefix}-${index}`)) index += 1
  return `${prefix}-${index}`
}

function sectionElement(section: ActiveSection) {
  if (!import.meta.client) return null
  return document.querySelector<HTMLElement>(`[data-section-key="${section}"]`)
}

function focusElement(element: HTMLElement | null) {
  if (!element) return
  element.scrollIntoView({ behavior: 'smooth', block: 'start' })
  element.focus({ preventScroll: true })
}

async function setActiveSection(section: ActiveSection) {
  activeSection.value = section
  await nextTick()
  focusElement(sectionElement(section))
}

function handleSectionTabChange(section: string) {
  if (!activeSectionValues.includes(section as ActiveSection)) return
  void setActiveSection(section as ActiveSection)
}

async function focusByKey(focusKey: string, fallbackSection: ActiveSection) {
  await nextTick()
  if (!import.meta.client) return
  const target = document.querySelector<HTMLElement>(`[data-focus-key="${focusKey}"]`)
  if (target) {
    target.scrollIntoView({ behavior: 'smooth', block: 'center' })
    target.focus({ preventScroll: true })
    return
  }
  focusElement(sectionElement(fallbackSection))
}

function resetBibleDraft() {
  if (!activeBible.value) return
  bibleDraft.value = cloneBible(activeBible.value)
  bibleSaveState.value = 'idle'
  characterSyncState.value = 'idle'
  generatedCharacters.value = []
}

function profileToDraftCharacter(profile: CharacterProfile, fallbackId: string): StoryBible['characters'][number] {
  return {
    id: profile.id?.trim() || fallbackId,
    name: profile.name.trim(),
    role: profile.role.trim(),
    desire: profile.desire.trim(),
    wound: profile.wound.trim(),
    secret: profile.secret?.trim() || '',
    summary: profile.summary?.trim(),
    metadata: profile.metadata
  }
}

function characterToProfile(character: StoryBible['characters'][number]): CharacterProfile {
  return {
    id: character.id,
    name: character.name,
    role: character.role,
    desire: character.desire,
    wound: character.wound,
    secret: character.secret,
    summary: character.summary,
    metadata: character.metadata
  }
}

function normalizedText(value?: string) {
  return (value || '').trim().toLowerCase()
}

function protagonistHintFromBible(bible: StoryBible) {
  return bible.source_seed?.metadata?.protagonist || bible.source_seed?.protagonist || bible.characters[0]?.name || ''
}

function looksLikeProtagonistPlaceholder(character: StoryBible['characters'][number], bible: StoryBible) {
  if (character.entity_id?.trim()) return false
  if (character.secret?.trim()) return false
  const role = normalizedText(character.role)
  const name = normalizedText(character.name)
  const hint = normalizedText(protagonistHintFromBible(bible))
  return role.includes('主角')
    || role.includes('关键人物')
    || role.includes('关键角色')
    || role.includes('protagonist')
    || role.includes('lead')
    || (Boolean(hint) && (name === hint || hint.includes(name) || name.includes(hint)))
}

function characterProfileBrief(character: CharacterProfile) {
  return [
    character.name,
    character.role,
    character.desire,
    character.wound,
    character.secret,
    character.summary
  ].map((item) => item?.trim()).filter(Boolean).join(' / ')
}

function buildCharacterGenerationBrief(mode: CharacterProfileMode, bible: StoryBible) {
  const protagonistHint = protagonistHintFromBible(bible)
  const usableCharacters = mode === 'protagonist'
    ? bible.characters.filter((character) => !looksLikeProtagonistPlaceholder(character, bible))
    : bible.characters
  return [
    mode === 'protagonist' ? t('projectOverview.characterGenerator.prompts.protagonist') : t('projectOverview.characterGenerator.prompts.character'),
    bible.premise ? `${t('projectOverview.fields.premise')}：${bible.premise}` : '',
    bible.themes.length ? `${t('projectOverview.fields.themes')}：${bible.themes.join('、')}` : '',
    bible.world_rules.length ? `${t('projectOverview.worldRules')}：${bible.world_rules.join('；')}` : '',
    protagonistHint ? `${t('projectNew.protagonist')}：${protagonistHint}` : '',
    usableCharacters.length ? `${t('projectOverview.characters')}：${usableCharacters.map(characterToProfile).map(characterProfileBrief).filter(Boolean).join('；')}` : ''
  ].filter(Boolean).join('\n')
}

function buildLocalCharacterProfile(mode: CharacterProfileMode, bible: StoryBible): CharacterProfile {
  const protagonistHint = protagonistHintFromBible(bible)
  const theme = bible.themes.find((item) => item.trim()) || t('projectOverview.characterGenerator.localFallback.theme')
  const rule = bible.world_rules.find((item) => item.trim()) || t('projectOverview.characterGenerator.localFallback.rule')
  if (mode === 'protagonist') {
    return {
      name: protagonistHint || t('projectOverview.characterGenerator.localFallback.protagonistName'),
      role: t('projectOverview.characterGenerator.localFallback.protagonistRole'),
      desire: t('projectOverview.characterGenerator.localFallback.protagonistDesire', { premise: bible.premise || theme }),
      wound: t('projectOverview.characterGenerator.localFallback.protagonistWound', { rule }),
      secret: t('projectOverview.characterGenerator.localFallback.protagonistSecret', { theme }),
      summary: buildCharacterGenerationBrief(mode, bible)
    }
  }
  return {
    name: t('projectOverview.characterGenerator.localFallback.supportingName', { count: bible.characters.length + 1 }),
    role: t('projectOverview.characterGenerator.localFallback.supportingRole'),
    desire: t('projectOverview.characterGenerator.localFallback.supportingDesire', { premise: bible.premise || theme }),
    wound: t('projectOverview.characterGenerator.localFallback.supportingWound', { rule }),
    secret: t('projectOverview.characterGenerator.localFallback.supportingSecret', { theme }),
    summary: buildCharacterGenerationBrief(mode, bible)
  }
}

function mergeGeneratedCharacters(profiles: CharacterProfile[], mode: CharacterProfileMode) {
  const draft = bibleDraft.value
  if (!draft) return
  const existingIds = draft.characters.map((character) => character.id)
  profiles
    .filter((profile) => profile.name.trim())
    .forEach((profile) => {
      const existingIndex = draft.characters.findIndex((character) => character.name.trim() === profile.name.trim())
      const placeholderIndex = mode === 'protagonist' && existingIndex < 0
        ? draft.characters.findIndex((character) => looksLikeProtagonistPlaceholder(character, draft))
        : -1
      const targetIndex = existingIndex >= 0 ? existingIndex : placeholderIndex
      const existingCharacter = targetIndex >= 0 ? draft.characters[targetIndex] : undefined
      const fallbackId = createUniqueId('character', existingIds)
      existingIds.push(fallbackId)
      const nextCharacter = profileToDraftCharacter(profile, existingCharacter?.id || fallbackId)
      if (existingCharacter && targetIndex >= 0) {
        draft.characters[targetIndex] = {
          ...existingCharacter,
          ...nextCharacter
        }
      } else {
        draft.characters.push(nextCharacter)
      }
    })
}

async function syncDraftCharacters(bible: StoryBible, options: { rethrow?: boolean } = {}) {
  characterSyncState.value = 'syncing'
  try {
    const result = await workspace.syncCharacters(projectId.value, bible)
    if (activeBible.value) {
      bibleDraft.value = cloneBible(activeBible.value)
    }
    characterSyncState.value = 'synced'
    return result
  } catch (error) {
    workspace.recordError(t('projectOverview.resultScopes.characterSync'), error)
    characterSyncState.value = 'failed'
    if (options.rethrow) throw error
    return null
  }
}

async function persistStoryBibleDraft(options: { syncCharactersAfterSave?: boolean } = {}) {
  if (!bibleDraft.value) return null
  bibleSaveState.value = 'saving'
  characterSyncState.value = 'idle'
  try {
    const result = await workspace.updateStoryBible(projectId.value, cloneBible(bibleDraft.value))
    bibleDraft.value = cloneBible(result.data)
    bibleSaveState.value = 'saved'
    if (options.syncCharactersAfterSave) {
      await syncDraftCharacters(result.data)
    }
    return result
  } catch (error) {
    workspace.recordError(t('projectOverview.resultScopes.storyBibleSave'), error)
    bibleSaveState.value = 'failed'
    return null
  }
}

async function saveStoryBible() {
  await persistStoryBibleDraft({ syncCharactersAfterSave: true })
}

async function generateCharacters(mode: CharacterProfileMode) {
  if (!bibleDraft.value) return
  characterGenerationState.value = 'generating'
  generatedCharacters.value = []
  try {
    generatedCharacters.value = [buildLocalCharacterProfile(mode, bibleDraft.value)]
    mergeGeneratedCharacters(generatedCharacters.value, mode)
    if (bibleDraft.value) {
      await persistStoryBibleDraft({ syncCharactersAfterSave: true })
    }
    characterGenerationState.value = 'idle'
    await setActiveSection('characters')
    const latestCharacter = generatedCharacters.value.find((character) => character.name.trim())
    if (latestCharacter) {
      const index = bibleDraft.value?.characters.findIndex((character) => character.name.trim() === latestCharacter.name.trim()) ?? -1
      if (index >= 0) await focusByKey(`character-${index}-name`, 'characters')
    }
  } catch (error) {
    workspace.recordError(t('projectOverview.resultScopes.characterGeneration'), error)
    characterGenerationState.value = 'failed'
  }
}

async function addTheme() {
  if (!bibleDraft.value) return
  const index = bibleDraft.value.themes.length
  bibleDraft.value.themes.push('')
  await setActiveSection('story')
  await focusByKey(`theme-${index}`, 'story')
}

function removeTheme(index: number) {
  bibleDraft.value?.themes.splice(index, 1)
}

async function addWorldRule() {
  if (!bibleDraft.value) return
  const index = bibleDraft.value.world_rules.length
  bibleDraft.value.world_rules.push('')
  await setActiveSection('story')
  await focusByKey(`world-rule-${index}`, 'story')
}

function removeWorldRule(index: number) {
  bibleDraft.value?.world_rules.splice(index, 1)
}

async function addCharacter() {
  if (!bibleDraft.value) return
  const index = bibleDraft.value.characters.length
  bibleDraft.value.characters.push({
    id: createUniqueId('character', bibleDraft.value.characters.map((character) => character.id)),
    name: '',
    role: '',
    desire: '',
    wound: '',
    secret: ''
  })
  await setActiveSection('characters')
  await focusByKey(`character-${index}-name`, 'characters')
}

function removeCharacter(index: number) {
  bibleDraft.value?.characters.splice(index, 1)
}

async function addForeshadow() {
  if (!bibleDraft.value) return
  const index = bibleDraft.value.foreshadows.length
  bibleDraft.value.foreshadows.push({
    id: createUniqueId('foreshadow', bibleDraft.value.foreshadows.map((item) => item.id)),
    title: '',
    planted_in: '',
    payoff_hint: '',
    status: 'planted'
  })
  await setActiveSection('story')
  await focusByKey(`foreshadow-${index}-title`, 'story')
}

function removeForeshadow(index: number) {
  bibleDraft.value?.foreshadows.splice(index, 1)
}

async function addChapter() {
  if (!bibleDraft.value) return
  const index = bibleDraft.value.chapters.length
  bibleDraft.value.chapters.push({
    id: createUniqueId('chapter', bibleDraft.value.chapters.map((chapter) => chapter.id)),
    title: '',
    status: 'planned',
    summary: ''
  })
  await setActiveSection('chapters')
  await focusByKey(`chapter-${index}-title`, 'chapters')
}

async function openChapterInEditor(chapter: StoryBible['chapters'][number], index: number) {
  if (!bibleDraft.value) return
  openingChapterId.value = chapter.id
  try {
    const saved = await persistStoryBibleDraft()
    const savedChapter = saved?.data.chapters.find((item) => item.id === chapter.id) || chapter
    const ensured = await workspace.ensureChapter(projectId.value, {
      chapter_id: savedChapter.id,
      number: index + 1,
      title: savedChapter.title || t('editor.defaults.titleForChapter', { id: savedChapter.id }),
      status: savedChapter.status || 'planned',
      summary: savedChapter.summary || undefined
    })
    await navigateTo(`/projects/${projectId.value}/editor?chapter=${encodeURIComponent(ensured.data.chapter.id)}`)
  } catch (error) {
    workspace.recordError(t('projectOverview.resultScopes.chapterEnsure'), error)
  } finally {
    openingChapterId.value = ''
  }
}

function removeChapter(index: number) {
  bibleDraft.value?.chapters.splice(index, 1)
}

function chapterStatusLabel(status: string) {
  return t(`status.chapter.${status}`)
}

function foreshadowStatusLabel(status: string) {
  return t(`status.foreshadow.${status}`)
}
</script>

<template>
  <div class="space-y-6">
    <SectionHeader
      :title="bible?.title || t('nav.project')"
      :description="bible?.premise || t('projectOverview.loadingDescription')"
    >
      <template #actions>
        <UiButton variant="outline" :disabled="loading[`bible:${projectId}`]" @click="workspace.loadStoryBible(projectId)">
          {{ t('actions.refresh') }}
        </UiButton>
        <UiButton v-if="bibleDraft" variant="outline" :disabled="bibleSaveState === 'saving'" @click="resetBibleDraft">
          {{ t('actions.reset') }}
        </UiButton>
        <UiButton v-if="bibleDraft" variant="archive" :disabled="bibleSaveState === 'saving'" @click="saveStoryBible">
          <Loader2 v-if="bibleSaveState === 'saving'" class="h-4 w-4 animate-spin" />
          <Save v-else class="h-4 w-4" />
          {{ bibleSaveState === 'saving' ? t('actions.saving') : t('projectOverview.saveStoryBible') }}
        </UiButton>
      </template>
    </SectionHeader>

    <StatusAlert :errors="errors" />
    <div class="flex flex-wrap gap-2">
      <UiBadge v-if="bibleSaveState === 'saved'" variant="success">
        <CheckCircle2 class="h-3 w-3" />
        {{ t('actions.saved') }}
      </UiBadge>
      <UiBadge v-if="bibleSaveState === 'failed'" variant="gold">
        <WifiOff class="h-3 w-3" />
        {{ t('apiError.saveFailed') }}
      </UiBadge>
    </div>

    <div v-if="!bibleDraft" class="grid gap-4 md:grid-cols-3">
      <UiCard v-for="item in 6" :key="item" class="h-40 animate-pulse bg-muted/50" />
    </div>

    <template v-else>
      <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_340px]">
        <div class="space-y-6">
          <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,320px)]">
            <UiCard class="p-4 sm:p-6">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectOverview.workspaceEyebrow') }}</p>
                  <h2 class="mt-2 text-xl font-semibold">{{ t('projectOverview.workspaceTitle') }}</h2>
                </div>
                <BookMarked class="h-5 w-5 text-muted-foreground" />
              </div>
              <div class="mt-5 space-y-4">
                <div v-for="item in projectSummary" :key="item.label" class="rounded-2xl border border-border bg-muted/35 p-4">
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ item.label }}</p>
                  <p class="mt-2 whitespace-pre-line break-words text-sm leading-6">{{ item.value }}</p>
                </div>
              </div>
            </UiCard>

            <UiCard class="p-4 sm:p-6">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectOverview.nextActionEyebrow') }}</p>
                  <h2 class="mt-2 text-lg font-semibold">{{ t('projectOverview.nextActionTitle') }}</h2>
                </div>
                <PenLine class="h-5 w-5 text-muted-foreground" />
              </div>
              <div class="mt-5 space-y-4">
                <div class="rounded-2xl border border-border bg-muted/35 p-4">
                  <p class="text-sm font-medium">{{ chapterProgressSummary?.nextChapter?.title || t('projectOverview.empty.chapters') }}</p>
                  <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ chapterProgressSummary?.nextChapter?.summary || t('common.emptyValue') }}</p>
                  <p class="mt-3 text-xs text-muted-foreground">
                    {{ t('projectOverview.chapterSummary', { active: chapterProgressSummary?.active || 0, total: chapterProgressSummary?.total || 0 }) }}
                  </p>
                </div>
                <div class="grid gap-3 sm:grid-cols-2">
                  <UiButton
                    class="w-full"
                    :disabled="Boolean(chapterProgressSummary?.nextChapter?.id && openingChapterId === chapterProgressSummary.nextChapter.id)"
                    @click="chapterProgressSummary?.nextChapter && openChapterInEditor(chapterProgressSummary.nextChapter, Math.max(0, bibleDraft.chapters.findIndex((chapter) => chapter.id === chapterProgressSummary?.nextChapter?.id)))"
                  >
                    <Loader2 v-if="chapterProgressSummary?.nextChapter?.id && openingChapterId === chapterProgressSummary.nextChapter.id" class="h-4 w-4 animate-spin" />
                    {{ t('projectOverview.continueWriting') }}
                    <ArrowRight class="h-4 w-4" />
                  </UiButton>
                  <UiButton variant="outline" class="w-full" @click="setActiveSection('chapters')">
                    <FileText class="h-4 w-4" />
                    {{ t('projectOverview.reviewChapters') }}
                  </UiButton>
                </div>
              </div>
            </UiCard>
          </div>

          <UiCard class="p-4 sm:p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 class="text-lg font-semibold">{{ t('projectOverview.prepTitle') }}</h2>
                <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('projectOverview.prepDescription') }}</p>
              </div>
              <UiButton variant="outline" :to="`/projects/${projectId}/graph`">
                <GitFork class="h-4 w-4" />
                {{ t('nav.graph') }}
              </UiButton>
            </div>
            <div class="mt-5 grid gap-4 md:grid-cols-3">
              <button
                v-for="card in prepCards"
                :key="card.key"
                type="button"
                class="rounded-2xl border border-border bg-muted/35 p-4 text-left transition-colors hover:bg-muted/60 focus-ring"
                @click="card.action()"
              >
                <p class="text-sm font-medium">{{ card.title }}</p>
                <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ card.description }}</p>
                <p class="mt-4 text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectOverview.prepCount', { count: card.count }) }}</p>
              </button>
            </div>
          </UiCard>

          <UiCard class="p-4 sm:p-5">
            <UiTabs :model-value="activeSection" :tabs="sectionTabs" class="w-full justify-start" @update:model-value="handleSectionTabChange" />
          </UiCard>

          <UiCard v-if="activeSection === 'overview'" data-section-key="overview" tabindex="-1" class="p-4 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background sm:p-6">
            <div class="grid gap-4 md:grid-cols-2">
              <div class="rounded-2xl border border-border bg-muted/35 p-4">
                <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectOverview.sections.story') }}</p>
                <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('projectOverview.sectionDescriptions.story') }}</p>
                <UiButton variant="outline" class="mt-4 w-full sm:w-auto" @click="setActiveSection('story')">
                  {{ t('projectOverview.openStorySetup') }}
                </UiButton>
              </div>
              <div class="rounded-2xl border border-border bg-muted/35 p-4">
                <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectOverview.sections.characters') }}</p>
                <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('projectOverview.sectionDescriptions.characters') }}</p>
                <UiButton variant="outline" class="mt-4 w-full sm:w-auto" @click="setActiveSection('characters')">
                  {{ t('projectOverview.openCharacters') }}
                </UiButton>
              </div>
            </div>
          </UiCard>

          <UiCard v-else-if="activeSection === 'story'" data-section-key="story" tabindex="-1" class="p-4 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background sm:p-6">
            <div class="grid min-w-0 gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,420px)]">
              <div>
                <div class="flex min-w-0 items-center gap-2">
                  <BookMarked class="h-5 w-5 text-muted-foreground" />
                  <h2 class="text-lg font-semibold">{{ t('projectOverview.storyBible') }}</h2>
                </div>
                <label class="mt-5 block space-y-2">
                  <span class="text-sm text-muted-foreground">{{ t('projectOverview.fields.premise') }}</span>
                  <UiTextarea v-model="bibleDraft.premise" :rows="5" />
                </label>
                <div class="mt-5 space-y-3">
                  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <p class="text-sm font-medium">{{ t('projectOverview.fields.themes') }}</p>
                    <UiButton size="sm" variant="outline" class="w-full sm:w-auto" @click="addTheme">
                      <Plus class="h-4 w-4" />
                      {{ t('actions.add') }}
                    </UiButton>
                  </div>
                  <div v-if="bibleDraft.themes.length === 0" class="rounded-xl border border-border bg-muted/35 p-4 text-sm text-muted-foreground">
                    {{ t('projectOverview.empty.themes') }}
                  </div>
                  <div v-for="(_theme, index) in bibleDraft.themes" :key="index" class="flex min-w-0 gap-2">
                    <UiInput v-model="bibleDraft.themes[index]" :data-focus-key="`theme-${index}`" :placeholder="t('projectOverview.placeholders.theme')" class="min-w-0 flex-1" />
                    <UiButton size="icon" variant="destructive" class="shrink-0" @click="removeTheme(index)">
                      <Trash2 class="h-4 w-4" />
                    </UiButton>
                  </div>
                </div>
              </div>

              <div class="space-y-6">
                <div>
                  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <div class="flex min-w-0 items-center gap-2">
                      <ShieldCheck class="h-5 w-5 shrink-0 text-muted-foreground" />
                      <h2 class="break-words text-lg font-semibold">{{ t('projectOverview.worldRules') }}</h2>
                    </div>
                    <UiButton size="sm" variant="outline" class="w-full sm:w-auto" @click="addWorldRule">
                      <Plus class="h-4 w-4" />
                      {{ t('actions.add') }}
                    </UiButton>
                  </div>
                  <div v-if="bibleDraft.world_rules.length === 0" class="mt-5 rounded-xl border border-border bg-muted/35 p-4 text-sm text-muted-foreground">
                    {{ t('projectOverview.empty.worldRules') }}
                  </div>
                  <div v-else class="mt-5 space-y-3">
                    <div v-for="(_rule, index) in bibleDraft.world_rules" :key="index" class="flex min-w-0 gap-2">
                      <UiTextarea v-model="bibleDraft.world_rules[index]" :data-focus-key="`world-rule-${index}`" :rows="3" :placeholder="t('projectOverview.placeholders.worldRule')" class="min-w-0 flex-1" />
                      <UiButton size="icon" variant="destructive" class="shrink-0" @click="removeWorldRule(index)">
                        <Trash2 class="h-4 w-4" />
                      </UiButton>
                    </div>
                  </div>
                </div>

                <div>
                  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <h2 class="break-words text-lg font-semibold">{{ t('projectOverview.foreshadowing') }}</h2>
                    <UiButton size="sm" variant="outline" class="w-full sm:w-auto" @click="addForeshadow">
                      <Plus class="h-4 w-4" />
                      {{ t('actions.add') }}
                    </UiButton>
                  </div>
                  <div v-if="bibleDraft.foreshadows.length === 0" class="mt-5 rounded-xl border border-border bg-muted/35 p-4 text-sm text-muted-foreground">
                    {{ t('projectOverview.empty.foreshadowing') }}
                  </div>
                  <div v-else class="mt-5 space-y-4">
                    <div v-for="(item, index) in bibleDraft.foreshadows" :key="item.id || index" class="min-w-0 rounded-xl border border-border p-3 sm:p-4">
                      <div class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start">
                        <div class="grid min-w-0 flex-1 gap-3 md:grid-cols-2">
                          <label class="space-y-2">
                            <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.foreshadowTitle') }}</span>
                            <UiInput v-model="item.title" :data-focus-key="`foreshadow-${index}-title`" />
                          </label>
                          <label class="space-y-2">
                            <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.foreshadowStatus') }}</span>
                            <UiSelect v-model="item.status" :options="foreshadowStatusOptions" />
                          </label>
                          <label class="space-y-2 sm:col-span-2">
                            <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.plantedIn') }}</span>
                            <UiInput v-model="item.planted_in" />
                          </label>
                          <label class="space-y-2 sm:col-span-2">
                            <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.payoffHint') }}</span>
                            <UiTextarea v-model="item.payoff_hint" :rows="3" />
                          </label>
                        </div>
                        <UiButton size="icon" variant="destructive" class="self-end sm:self-start" @click="removeForeshadow(index)">
                          <Trash2 class="h-4 w-4" />
                        </UiButton>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </UiCard>

          <UiCard v-else-if="activeSection === 'characters'" data-section-key="characters" tabindex="-1" class="p-4 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background sm:p-6">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="flex min-w-0 items-center gap-2">
                <UserRound class="h-5 w-5 shrink-0 text-muted-foreground" />
                <h2 class="break-words text-lg font-semibold">{{ t('projectOverview.characters') }}</h2>
              </div>
              <div class="flex flex-wrap gap-2">
                <UiBadge v-if="characterSyncState === 'syncing'" variant="gold">
                  <Loader2 class="h-3 w-3 animate-spin" />
                  {{ t('projectOverview.characterSync.syncing') }}
                </UiBadge>
                <UiBadge v-else-if="characterSyncState === 'synced'" variant="success">
                  <CheckCircle2 class="h-3 w-3" />
                  {{ t('projectOverview.characterSync.synced') }}
                </UiBadge>
                <UiBadge v-else-if="characterSyncState === 'failed'" variant="rose">
                  <WifiOff class="h-3 w-3" />
                  {{ t('projectOverview.characterSync.failed') }}
                </UiBadge>
                <UiButton size="sm" variant="outline" class="w-full sm:w-auto" @click="addCharacter">
                  <Plus class="h-4 w-4" />
                  {{ t('actions.add') }}
                </UiButton>
              </div>
            </div>

            <div class="mt-5 rounded-2xl border border-border bg-muted/25 p-4">
              <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                <div>
                  <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectOverview.characterGenerator.eyebrow') }}</p>
                  <h3 class="mt-2 font-semibold">{{ t('projectOverview.characterGenerator.title') }}</h3>
                  <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t('projectOverview.characterGenerator.description') }}</p>
                </div>
                <div class="flex flex-wrap gap-2">
                  <UiButton size="sm" variant="outline" :disabled="characterGenerationState === 'generating'" @click="generateCharacters('protagonist')">
                    <Loader2 v-if="characterGenerationState === 'generating'" class="h-4 w-4 animate-spin" />
                    <Sparkles v-else class="h-4 w-4" />
                    {{ t('projectOverview.characterGenerator.generateProtagonist') }}
                  </UiButton>
                  <UiButton size="sm" :disabled="characterGenerationState === 'generating'" @click="generateCharacters('character')">
                    <Loader2 v-if="characterGenerationState === 'generating'" class="h-4 w-4 animate-spin" />
                    <UserRound v-else class="h-4 w-4" />
                    {{ t('projectOverview.characterGenerator.generateCharacter') }}
                  </UiButton>
                </div>
              </div>
              <div v-if="generatedCharacters.length" class="mt-4 rounded-xl border border-border bg-card/80 p-3">
                <p class="text-sm font-medium text-foreground">{{ t('projectOverview.characterGenerator.latestResult') }}</p>
                <ul class="mt-2 space-y-2 text-sm leading-6">
                  <li v-for="character in generatedCharacters" :key="character.name" class="break-words">
                    {{ [character.name, character.role, character.desire].filter(Boolean).join(' · ') }}
                  </li>
                </ul>
              </div>
              <div v-else-if="characterGenerationState === 'failed'" class="mt-4 rounded-xl border border-destructive/30 bg-destructive/10 px-3 py-3 text-sm text-destructive">
                {{ t('projectOverview.characterGenerator.failed') }}
              </div>
            </div>

            <div v-if="bibleDraft.characters.length === 0" class="mt-5 rounded-xl border border-border bg-muted/35 p-4 text-sm text-muted-foreground">
              {{ t('projectOverview.empty.characters') }}
            </div>
            <div v-else class="mt-5 space-y-4">
              <div v-for="(character, index) in bibleDraft.characters" :key="character.id || index" class="min-w-0 rounded-xl border border-border p-3 sm:p-4">
                <div class="mb-3 flex flex-wrap items-center gap-2">
                  <UiBadge v-if="character.entity_id" variant="success">{{ t('projectOverview.characterSync.realCharacter') }}</UiBadge>
                  <UiBadge v-else variant="muted">{{ t('projectOverview.characterSync.pending') }}</UiBadge>
                  <UiBadge v-if="character.sync_status" variant="muted">{{ character.sync_status }}</UiBadge>
                </div>
                <div class="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div class="grid min-w-0 flex-1 gap-3 md:grid-cols-2">
                    <label class="space-y-2">
                      <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.characterName') }}</span>
                      <UiInput v-model="character.name" :data-focus-key="`character-${index}-name`" />
                    </label>
                    <label class="space-y-2">
                      <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.characterRole') }}</span>
                      <UiInput v-model="character.role" />
                    </label>
                    <label class="space-y-2 sm:col-span-2">
                      <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.characterDesire') }}</span>
                      <UiTextarea v-model="character.desire" :rows="3" />
                    </label>
                    <label class="space-y-2 sm:col-span-2">
                      <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.characterWound') }}</span>
                      <UiTextarea v-model="character.wound" :rows="3" />
                    </label>
                    <label class="space-y-2 sm:col-span-2">
                      <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.characterSecret') }}</span>
                      <UiInput v-model="character.secret" />
                    </label>
                    <label class="space-y-2 sm:col-span-2">
                      <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.characterSummary') }}</span>
                      <UiTextarea v-model="character.summary" :rows="3" />
                    </label>
                  </div>
                  <UiButton size="icon" variant="destructive" class="self-end sm:self-start" @click="removeCharacter(index)">
                    <Trash2 class="h-4 w-4" />
                  </UiButton>
                </div>
              </div>
            </div>
          </UiCard>

          <UiCard v-else data-section-key="chapters" tabindex="-1" class="p-4 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background sm:p-6">
            <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <h2 class="break-words text-lg font-semibold">{{ t('projectOverview.chapters') }}</h2>
              <div class="grid gap-2 sm:flex sm:flex-wrap">
                <UiButton variant="outline" class="w-full sm:w-auto" @click="addChapter"><Plus class="h-4 w-4" />{{ t('actions.add') }}</UiButton>
                <UiButton class="w-full sm:w-auto" :to="`/projects/${projectId}/editor`">{{ t('nav.editor') }}<ArrowRight class="h-4 w-4" /></UiButton>
              </div>
            </div>
            <div v-if="bibleDraft.chapters.length === 0" class="mt-5 rounded-xl border border-border bg-muted/35 p-4 text-sm text-muted-foreground">
              {{ t('projectOverview.empty.chapters') }}
            </div>
            <div v-else class="mt-5 grid gap-4 xl:grid-cols-2">
              <div
                v-for="(chapter, index) in chapterCards"
                :key="chapter.id || index"
                class="min-w-0 rounded-xl border border-border p-3 sm:p-4"
              >
                <div class="flex min-w-0 flex-col gap-3">
                  <div class="grid min-w-0 gap-3 sm:grid-cols-2">
                    <label class="space-y-2">
                      <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.chapterTitle') }}</span>
                      <UiInput v-model="chapter.title" :data-focus-key="`chapter-${index}-title`" />
                    </label>
                    <label class="space-y-2">
                      <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.chapterStatus') }}</span>
                      <UiSelect v-model="chapter.status" :options="chapterStatusOptions" />
                    </label>
                    <label class="space-y-2 sm:col-span-2">
                      <span class="text-xs text-muted-foreground">{{ t('projectOverview.fields.chapterSummary') }}</span>
                      <UiTextarea v-model="chapter.summary" :rows="4" />
                    </label>
                  </div>
                  <div class="flex min-w-0 flex-wrap items-center justify-between gap-3">
                    <UiButton
                      type="button"
                      variant="outline"
                      class="min-w-0 flex-1 justify-between"
                      :disabled="openingChapterId === chapter.id"
                      @click="openChapterInEditor(chapter, index)"
                    >
                      <Loader2 v-if="openingChapterId === chapter.id" class="h-4 w-4 animate-spin" />
                      <span>{{ t('projectOverview.openChapter') }}</span>
                      <ArrowRight class="h-4 w-4" />
                    </UiButton>
                    <UiButton size="icon" variant="destructive" class="shrink-0" @click="removeChapter(index)">
                      <Trash2 class="h-4 w-4" />
                    </UiButton>
                  </div>
                </div>
              </div>
            </div>
          </UiCard>
        </div>

        <aside class="space-y-6">
          <UiCard class="p-4 sm:p-6">
            <p class="text-xs uppercase tracking-[0.18em] text-muted-foreground">{{ t('projectOverview.quickActions') }}</p>
            <div class="mt-4 grid gap-3">
              <UiButton
                class="w-full justify-between"
                :disabled="Boolean(chapterProgressSummary?.nextChapter?.id && openingChapterId === chapterProgressSummary.nextChapter.id)"
                @click="chapterProgressSummary?.nextChapter && openChapterInEditor(chapterProgressSummary.nextChapter, Math.max(0, bibleDraft.chapters.findIndex((chapter) => chapter.id === chapterProgressSummary?.nextChapter?.id)))"
              >
                <span class="inline-flex items-center gap-2">
                  <Loader2 v-if="chapterProgressSummary?.nextChapter?.id && openingChapterId === chapterProgressSummary.nextChapter.id" class="h-4 w-4 animate-spin" />
                  {{ t('projectOverview.continueWriting') }}
                </span>
                <ArrowRight class="h-4 w-4" />
              </UiButton>
              <UiButton variant="outline" class="w-full justify-between" @click="setActiveSection('story')">
                <span>{{ t('projectOverview.editStoryBible') }}</span>
                <BookMarked class="h-4 w-4" />
              </UiButton>
              <UiButton variant="outline" class="w-full justify-between" @click="setActiveSection('chapters')">
                <span>{{ t('projectOverview.planChapters') }}</span>
                <FilePenLine class="h-4 w-4" />
              </UiButton>
            </div>
          </UiCard>
        </aside>
      </div>
    </template>
  </div>
</template>
