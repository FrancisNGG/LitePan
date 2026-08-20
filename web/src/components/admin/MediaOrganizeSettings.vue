<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { getApiErrorMessage } from "@/api/client";
import {
  fetchMediaOrganizeSettings,
  saveMediaOrganizeSettings,
  testMediaOrganizeTmdb,
  testMediaOrganizeTemplates,
  type MediaOrganizeSettings,
} from "@/api/mediaOrganize";
import AppButton from "@/components/base/AppButton.vue";
import AppInput from "@/components/base/AppInput.vue";
import AppSelect from "@/components/base/AppSelect.vue";
import SettingsBoolSegment from "@/components/admin/SettingsBoolSegment.vue";
import SettingsCard from "@/components/admin/SettingsCard.vue";
import SettingsHelpTooltip from "@/components/admin/SettingsHelpTooltip.vue";
import SettingsRow from "@/components/admin/SettingsRow.vue";
import TmdbHostsHelpTip from "@/components/admin/TmdbHostsHelpTip.vue";
import { useSettingsForm, bindSettingsPanelExpose } from "@/composables/useSettingsForm";
import { useSettingsLoad } from "@/composables/useSettingsLoad";
import { toast } from "@/composables/useToast";
import "@/styles/admin-shared.css";

const ORGANIZE_SETTINGS_ACCENT = "#10b981";
const ALL_TAG_KEYS = ["screen_size", "frame_rate", "video_codec", "audio_codec", "audio_channels"] as const;
const TAG_LABELS: Record<string, string> = {
  screen_size: "分辨率",
  frame_rate: "帧率",
  video_codec: "视频编码",
  audio_codec: "音频编码",
  audio_channels: "声道数",
};

const tmdbLanguageOptions = [
  { value: "zh-CN", label: "简体中文" },
  { value: "zh-TW", label: "繁体中文" },
  { value: "en-US", label: "English" },
];

const conflictPolicyOptions = [
  { value: "skip", label: "跳过（推荐）" },
  { value: "overwrite", label: "覆盖" },
];

const { loading, loaded, runLoad } = useSettingsLoad();
const saving = ref(false);
const tmdbTesting = ref(false);
const draggingTagIndex = ref<number | null>(null);
const insertIndex = ref<number | null>(null);
const tagEditorRef = ref<HTMLElement | null>(null);

const tagGhost = reactive({
  visible: false,
  x: 0,
  y: 0,
  width: 0,
  height: 0,
  label: "",
});

let pendingTagDrag: {
  index: number;
  startX: number;
  startY: number;
  offsetX: number;
  offsetY: number;
  width: number;
  height: number;
} | null = null;

const tagGhostStyle = computed(() => ({
  left: `${tagGhost.x}px`,
  top: `${tagGhost.y}px`,
  width: `${tagGhost.width}px`,
  height: `${tagGhost.height}px`,
}));

const {
  settings,
  isDirty: settingsChanged,
  isFieldChanged,
  snapshotBaseline,
  revert: revertToBaseline,
} = useSettingsForm<MediaOrganizeSettings>({
  proxy_enabled: false,
  proxy_url: "",
  proxy_username: "",
  proxy_password: "",
  tmdb_api_key: "",
  tmdb_language: "zh-CN",
  api_request_interval_ms: 300,
  tmdb_request_interval_ms: 250,
  file_extensions: "",
  metadata_extensions: "",
  media_tag_order: "",
  align_media_tags: false,
  max_works_per_run: 50,
  overwrite_existing: false,
  movie_naming_format: `{% if title %}{{ title }}{% endif %}{% if en_title %} {{ en_title }}{% endif %}{% if year %} ({{ year }}){% endif %}/{% if title %}{{ title }}{% endif %}{% if en_title %} {{ en_title }}{% endif %}{% if year %} ({{ year }}){% endif %}{% if part %}-{{ part }}{% endif %} - [{% set has_any = false %}{% if webSource %}{{ webSource }}{% set has_any = true %}{% endif %}{% if resourceType %}{% if has_any %} {% endif %}{{ resourceType }}{% set has_any = true %}{% endif %}{% if effect %}{% if has_any %} {% endif %}{{ effect }}{% set has_any = true %}{% endif %}{% if ('remux' in original_name.lower()) %}{% if has_any %} {% endif %}Remux{% set has_any = true %}{% endif %}{% if videoFormat %}{% if has_any %} {% endif %}{{ videoFormat }}{% set has_any = true %}{% endif %}{% if videoCodec %}{% if has_any %} {% endif %}{{ videoCodec }}{% set has_any = true %}{% endif %}{% if audioCodec %}{% if has_any %} {% endif %}{{ audioCodec }}{% set has_any = true %}{% endif %}{% if releaseGroup %}{% if has_any %} {% endif %}{{ releaseGroup }}{% set has_any = true %}{% endif %}]{% if fileExt %}{{ fileExt }}{% endif %}`,
  tv_naming_format: `{% if title %}{{ title }}{% endif %}{% if en_title %} {{ en_title }}{% endif %}{% if year %} ({{year}}){% endif %}/Season {{season}}/{% if title %}{{ title }}{% endif %}{% if en_title %} {{ en_title }}{% endif %} - {{season_episode}}{% if part %}-{{part}}{% endif %}{% if episode %} - [{% set has_any = false %}{% if webSource %}{{ webSource }}{% set has_any = true %}{% endif %}{% if resourceType %}{% if has_any %} {% endif %}{{ resourceType }}{% set has_any = true %}{% endif %}{% if effect %}{% if has_any %} {% endif %}{{ effect }}{% set has_any = true %}{% endif %}{% if ('remux' in original_name.lower()) %}{% if has_any %} {% endif %}Remux{% set has_any = true %}{% endif %}{% if videoFormat %}{% if has_any %} {% endif %}{{ videoFormat }}{% set has_any = true %}{% endif %}{% if videoCodec %}{% if has_any %} {% endif %}{{ videoCodec }}{% set has_any = true %}{% endif %}{% if audioCodec %}{% if has_any %} {% endif %}{{ audioCodec }}{% set has_any = true %}{% endif %}{% if releaseGroup %}{% if has_any %} {% endif %}{{ releaseGroup }}{% set has_any = true %}{% endif %}] - 第 {{episode}} 集{% endif %}{% if fileExt %}{{ fileExt }}{% endif %}`,
  category_map: `{"movie":{"动画电影":{"genre_ids":"16"},"华语电影":{"original_language":"zh,cn,bo,za"},"外语电影":{}},"tv":{"国漫":{"genre_ids":"16","origin_country":"CN,TW,HK"},"日番":{"genre_ids":"16","origin_country":"JP"},"纪录片":{"genre_ids":"99"},"儿童":{"genre_ids":"10762"},"综艺":{"genre_ids":"10764,10767"},"国产剧":{"origin_country":"CN,TW,HK"},"欧美剧":{"origin_country":"US,FR,GB,DE,ES,IT,NL,PT,RU,UK"},"日韩剧":{"origin_country":"JP,KP,KR,TH,IN,SG"},"未分类":{}}}`,
  release_groups: "",
});
const tagOrder = reactive<string[]>([...ALL_TAG_KEYS]);

const disabledTags = computed(() => ALL_TAG_KEYS.filter((k) => !tagOrder.includes(k)));

const conflictPolicy = computed({
  get: () => (settings.overwrite_existing ? "overwrite" : "skip"),
  set: (val: string) => {
    settings.overwrite_existing = val === "overwrite";
  },
});

function parseMediaTagOrder(raw: MediaOrganizeSettings["media_tag_order"]): string[] | null {
  if (Array.isArray(raw)) return raw.filter((k) => ALL_TAG_KEYS.includes(k as (typeof ALL_TAG_KEYS)[number]));
  if (typeof raw === "string" && raw.trim()) {
    try {
      let parsed: unknown = JSON.parse(raw);
      if (typeof parsed === "string") parsed = JSON.parse(parsed);
      if (Array.isArray(parsed)) {
        return parsed.filter((k) => ALL_TAG_KEYS.includes(k as (typeof ALL_TAG_KEYS)[number]));
      }
    } catch {
      return null;
    }
  }
  return null;
}

function syncTagsFromSettings() {
  const order = parseMediaTagOrder(settings.media_tag_order);
  tagOrder.splice(0, tagOrder.length, ...(order ?? [...ALL_TAG_KEYS]));
}

function flushTagOrderToSettings() {
  settings.media_tag_order = JSON.stringify([...tagOrder]);
}

function removeTag(key: string) {
  const idx = tagOrder.indexOf(key);
  if (idx >= 0) tagOrder.splice(idx, 1);
  flushTagOrderToSettings();
}

function addTag(key: string) {
  if (!tagOrder.includes(key)) tagOrder.push(key);
  flushTagOrderToSettings();
}

function startTagPointerDrag(index: number, e: PointerEvent) {
  if (e.button !== 0 || draggingTagIndex.value !== null) return;
  if ((e.target as HTMLElement | null)?.closest(".tag-chip__remove")) return;
  const chip = e.currentTarget as HTMLElement;
  const rect = chip.getBoundingClientRect();
  pendingTagDrag = {
    index,
    startX: e.clientX,
    startY: e.clientY,
    offsetX: e.clientX - rect.left,
    offsetY: e.clientY - rect.top,
    width: rect.width,
    height: rect.height,
  };
  document.addEventListener("pointermove", handleTagPointerMove);
  document.addEventListener("pointerup", finishTagPointerDrag);
  document.addEventListener("pointercancel", cancelTagPointerDrag);
}

function beginTagDrag(e: PointerEvent) {
  if (!pendingTagDrag) return;
  const index = pendingTagDrag.index;
  draggingTagIndex.value = index;
  insertIndex.value = index;
  tagGhost.visible = true;
  tagGhost.label = TAG_LABELS[tagOrder[index]] ?? tagOrder[index];
  tagGhost.width = pendingTagDrag.width;
  tagGhost.height = pendingTagDrag.height;
  tagGhost.x = e.clientX - pendingTagDrag.offsetX;
  tagGhost.y = e.clientY - pendingTagDrag.offsetY;
  document.body.classList.add("mo-tag-dragging");
}

function handleTagPointerMove(e: PointerEvent) {
  if (!pendingTagDrag) return;
  const dx = Math.abs(e.clientX - pendingTagDrag.startX);
  const dy = Math.abs(e.clientY - pendingTagDrag.startY);
  if (draggingTagIndex.value === null) {
    if (dx < 4 && dy < 4) return;
    beginTagDrag(e);
  }
  e.preventDefault();
  tagGhost.x = e.clientX - pendingTagDrag.offsetX;
  tagGhost.y = e.clientY - pendingTagDrag.offsetY;
  updateInsertIndex(e.clientX, e.clientY);
}

function updateInsertIndex(clientX: number, clientY: number) {
  const root = tagEditorRef.value;
  if (!root || draggingTagIndex.value === null) return;

  const chips = Array.from(root.querySelectorAll<HTMLElement>(".tag-chip[data-tag-index]"));
  if (!chips.length) {
    insertIndex.value = 0;
    return;
  }

  const rowChips = chips.filter((el) => {
    const rect = el.getBoundingClientRect();
    return clientY >= rect.top - 12 && clientY <= rect.bottom + 12;
  });
  const targets = (rowChips.length ? rowChips : chips).sort(
    (a, b) => a.getBoundingClientRect().left - b.getBoundingClientRect().left,
  );

  for (const el of targets) {
    const rect = el.getBoundingClientRect();
    const tagIndex = Number(el.dataset.tagIndex);
    if (Number.isNaN(tagIndex)) continue;
    if (clientX < rect.left + rect.width / 2) {
      insertIndex.value = tagIndex;
      return;
    }
  }

  const lastIndex = Number(targets[targets.length - 1].dataset.tagIndex);
  insertIndex.value = Number.isNaN(lastIndex) ? tagOrder.length : lastIndex + 1;
}

function applyTagMove(from: number, insert: number) {
  if (insert === from || insert === from + 1) return;
  const item = tagOrder.splice(from, 1)[0];
  const target = from < insert ? insert - 1 : insert;
  tagOrder.splice(target, 0, item);
  flushTagOrderToSettings();
}

function finishTagPointerDrag() {
  if (draggingTagIndex.value !== null && insertIndex.value !== null) {
    applyTagMove(draggingTagIndex.value, insertIndex.value);
  }
  endTagDrag();
  cleanupTagPointerListeners();
}

function cancelTagPointerDrag() {
  endTagDrag();
  cleanupTagPointerListeners();
}

function cleanupTagPointerListeners() {
  pendingTagDrag = null;
  document.removeEventListener("pointermove", handleTagPointerMove);
  document.removeEventListener("pointerup", finishTagPointerDrag);
  document.removeEventListener("pointercancel", cancelTagPointerDrag);
}

function endTagDrag() {
  draggingTagIndex.value = null;
  insertIndex.value = null;
  tagGhost.visible = false;
  document.body.classList.remove("mo-tag-dragging");
}

onBeforeUnmount(() => {
  cleanupTagPointerListeners();
  endTagDrag();
});

async function loadSettings(options?: { silent?: boolean }) {
  await runLoad(async () => {
    const data = await fetchMediaOrganizeSettings();
    Object.assign(settings, data);
    if (settings.max_works_per_run == null) settings.max_works_per_run = 50;
    syncTagsFromSettings();
    snapshotBaseline();
  }, "加载整理设置失败", options);
}

async function saveSettings() {
  if (!settingsChanged.value) return;
  saving.value = true;
  try {
    flushTagOrderToSettings();
    const data = await saveMediaOrganizeSettings({ ...settings });
    Object.assign(settings, data);
    syncTagsFromSettings();
    snapshotBaseline();
    toast.success("整理设置已保存");
  } catch (e) {
    toast.error(getApiErrorMessage(e, "保存整理设置失败"));
  } finally {
    saving.value = false;
  }
}

async function testTmdb() {
  tmdbTesting.value = true;
  try {
    const result = await testMediaOrganizeTmdb({
      tmdb_api_key: settings.tmdb_api_key,
      tmdb_language: settings.tmdb_language,
      proxy_enabled: settings.proxy_enabled,
      proxy_url: settings.proxy_url,
      proxy_username: settings.proxy_username,
      proxy_password: settings.proxy_password,
    });
    if (result.ok) {
      toast.success(`TMDB 连通正常（语言：${result.language ?? settings.tmdb_language}）。请确认已点击「保存设置」，再重新生成整理计划。`);
    } else {
      toast.error("TMDB 连通测试失败");
    }
  } catch (e) {
    toast.error(getApiErrorMessage(e, "TMDB 测试失败"));
  } finally {
    tmdbTesting.value = false;
  }
}

onMounted(() => {
  void loadSettings();
});

function revertPanelSettings() {
  revertToBaseline();
  syncTagsFromSettings();
}

const tplTest = reactive({
  filename: "流浪地球.2019.2160p.mkv",
});
const tplTesting = ref(false);
const tplResult = ref<{ full_path?: string; is_tv?: boolean; category?: string } | null>(null);

async function testTemplates() {
  tplTesting.value = true;
  tplResult.value = null;
  try {
    tplResult.value = await testMediaOrganizeTemplates({
      movie_naming_template: settings.movie_naming_format,
      tv_naming_template: settings.tv_naming_format,
      filename: tplTest.filename,
    });
  } catch (e) {
    toast.error(getApiErrorMessage(e, "模板测试失败"));
  } finally {
    tplTesting.value = false;
  }
}

defineExpose(
  bindSettingsPanelExpose({
    isDirty: settingsChanged,
    saving,
    save: saveSettings,
    reload: () => loadSettings({ silent: loaded.value }),
    revert: revertPanelSettings,
  }),
);
</script>

<template>
  <div class="mo-settings">
    <div v-if="loading" class="settings-card__loading">加载中…</div>

    <template v-else>
      <SettingsCard title="代理设置" :accent="ORGANIZE_SETTINGS_ACCENT">
        <template #head-aside>
          <TmdbHostsHelpTip />
        </template>
        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('proxy_enabled')">
          <template #info>
            <div class="settings-row__label"><span>启用代理</span></div>
          </template>
          <template #control>
            <SettingsBoolSegment v-model="settings.proxy_enabled" label="启用代理" />
          </template>
        </SettingsRow>

        <template v-if="settings.proxy_enabled">
          <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('proxy_url')">
            <template #info>
              <div class="settings-row__label"><span>代理地址</span></div>
            </template>
            <template #control>
              <AppInput v-model="settings.proxy_url" placeholder="http://127.0.0.1:1080 或 socks5://127.0.0.1:1080" />
            </template>
          </SettingsRow>

          <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('proxy_username')">
            <template #info>
              <div class="settings-row__label"><span>代理用户名</span></div>
            </template>
            <template #control>
              <AppInput
                v-model="settings.proxy_username"
                autocomplete="off"
                placeholder="可选"
              />
            </template>
          </SettingsRow>

          <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('proxy_password')">
            <template #info>
              <div class="settings-row__label"><span>代理密码</span></div>
            </template>
            <template #control>
              <AppInput v-model="settings.proxy_password" type="password" autocomplete="new-password" placeholder="可选" />
            </template>
          </SettingsRow>
        </template>
      </SettingsCard>

      <SettingsCard title="TMDB 设置" :accent="ORGANIZE_SETTINGS_ACCENT">
        <template #head-actions>
          <AppButton type="button" variant="secondary" size="sm" :disabled="tmdbTesting" @click="testTmdb">
            {{ tmdbTesting ? "测试中…" : "测试连通性" }}
          </AppButton>
        </template>

        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('tmdb_api_key')">
          <template #info>
            <div class="settings-row__label"><span>TMDB API Key</span></div>
          </template>
          <template #control>
            <AppInput
              v-model="settings.tmdb_api_key"
              type="password"
              placeholder="请填写 TMDB API Key（必填）"
              :ignore-autofill="true"
            />
          </template>
        </SettingsRow>

        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('tmdb_language')">
          <template #info>
            <div class="settings-row__label"><span>TMDB 语言（影响搜索和命名）</span></div>
          </template>
          <template #control>
            <AppSelect v-model="settings.tmdb_language" :options="tmdbLanguageOptions" />
          </template>
        </SettingsRow>
      </SettingsCard>

      <SettingsCard title="API 请求节流" :accent="ORGANIZE_SETTINGS_ACCENT">
        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('api_request_interval_ms')">
          <template #info>
            <div class="settings-row__label"><span>API 额外补偿间隔（毫秒）</span></div>
          </template>
          <template #control>
            <AppInput v-model="settings.api_request_interval_ms" type="number" min="100" max="10000" />
          </template>
        </SettingsRow>

        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('tmdb_request_interval_ms')">
          <template #info>
            <div class="settings-row__label"><span>TMDB 请求间隔（毫秒）</span></div>
          </template>
          <template #control>
            <AppInput v-model="settings.tmdb_request_interval_ms" type="number" min="100" max="5000" />
          </template>
        </SettingsRow>
      </SettingsCard>

      <SettingsCard title="文件识别与整理规则" :accent="ORGANIZE_SETTINGS_ACCENT">
        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('file_extensions')">
          <template #info>
            <div class="settings-row__label"><span>媒体文件后缀（分号分隔）</span></div>
          </template>
          <template #control>
            <AppInput v-model="settings.file_extensions" placeholder="mkv;mp4;avi;ts;mov…" />
          </template>
        </SettingsRow>

        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('metadata_extensions')">
          <template #info>
            <div class="settings-row__label"><span>元数据文件后缀（分号分隔）</span></div>
          </template>
          <template #control>
            <AppInput v-model="settings.metadata_extensions" placeholder="nfo;ass;srt;sub…" />
          </template>
        </SettingsRow>

        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('max_works_per_run')">
          <template #info>
            <div class="settings-row__label">
              <span>每次最多整理作品数</span>
              <SettingsHelpTooltip title="分批整理说明">
                <p>每次生成计划最多包含这么多部作品（一部电影或一部剧集算 1 部），达到上限后停止扫描。</p>
                <p>已整理过的（带 tmdb 标识）不计入此数。</p>
                <p>执行完后再次生成计划即可处理剩余作品。0 表示不限制（不推荐用于大库）。</p>
              </SettingsHelpTooltip>
            </div>
          </template>
          <template #control>
            <AppInput v-model="settings.max_works_per_run" type="number" min="0" max="10000" placeholder="50" />
          </template>
        </SettingsRow>

        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('overwrite_existing')">
          <template #info>
            <div class="settings-row__label">
              <span>同名冲突处理</span>
              <SettingsHelpTooltip title="同名冲突处理说明">
                <p>执行整理时，若目标目录已存在同名文件：</p>
                <p><b>跳过</b>：保留目标已有文件，跳过该项（推荐，更安全）</p>
                <p><b>覆盖</b>：先删除目标已有同名文件，再写入新文件</p>
              </SettingsHelpTooltip>
            </div>
          </template>
          <template #control>
            <AppSelect v-model="conflictPolicy" :options="conflictPolicyOptions" />
          </template>
        </SettingsRow>
      </SettingsCard>

      <SettingsCard title="命名模板（MoviePilot）" :accent="ORGANIZE_SETTINGS_ACCENT">
        <template #head-aside>
          <span class="mo-tpl-tip" v-text="'Jinja2 完整路径模板，渲染结果含目录层级，按 / 自动拆分为作品目录/季目录/文件名'"></span>
        </template>
        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('movie_naming_format')">
          <template #info>
            <div class="settings-row__label">
              <span>电影命名模板</span>
              <SettingsHelpTooltip title="电影命名模板说明">
                <p>MoviePilot 兼容的 Jinja2 模板，渲染<code>完整路径</code>（目录/文件名）。</p>
                <p>示例：<code>{{ '{' }}{{ '{' }}title{{ '}' }}{% if year %} ({{ '{' }}{{ '{' }}year{{ '}' }}){% endif %}/{{ '{' }}{{ '{' }}title{{ '}' }}{% if year %} ({{ '{' }}{{ '{' }}year{{ '}' }}){% endif %}{% if part %}-{{ '{' }}{{ '{' }}part{{ '}' }}{% endif %}{% if videoFormat %} - {{ '{' }}{{ '{' }}videoFormat{{ '}' }}{% endif %}{{ '{' }}{{ '{' }}fileExt{{ '}' }}{{ '}' }}</code></p>
                <p>渲染示例：<code>流浪地球 (2019)/流浪地球 (2019) - 2160p.mkv</code></p>
                <p>留空则使用任务级模板（作品文件夹/文件名模板）。</p>
              </SettingsHelpTooltip>
            </div>
          </template>
          <template #control>
            <AppInput v-model="settings.movie_naming_format" placeholder="{{ '{' }}{{ '{' }}title{{ '}' }}{% if year %} ({{ '{' }}{{ '{' }}year{{ '}' }}){% endif %}/{{ '{' }}{{ '{' }}title{{ '}' }}…" />
          </template>
        </SettingsRow>
        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('tv_naming_format')">
          <template #info>
            <div class="settings-row__label">
              <span>电视剧命名模板</span>
              <SettingsHelpTooltip title="电视剧命名模板说明">
                <p>MoviePilot 兼容的 Jinja2 模板，渲染<code>完整路径</code>（作品目录/季目录/文件名）。</p>
                <p>示例：<code>{{ '{' }}{{ '{' }}title{{ '}' }}{% if year %} ({{ '{' }}{{ '{' }}year{{ '}' }}){% endif %}/Season {{ '{' }}{{ '{' }}season{{ '}' }}/{{ '{' }}{{ '{' }}title{{ '}' }} - {{ '{' }}{{ '{' }}season_episode{{ '}' }}{{ '}' }}{% if episode %} - 第 {{ '{' }}{{ '{' }}episode{{ '}' }} 集{% endif %}{{ '{' }}{{ '{' }}fileExt{{ '}' }}{{ '}' }}</code></p>
                <p>渲染示例：<code>三体 (2023)/Season 1/三体 - S01E02 - 第 2 集.mkv</code></p>
                <p>留空则使用任务级模板（季文件夹/作品文件夹/文件名模板）。</p>
              </SettingsHelpTooltip>
            </div>
          </template>
          <template #control>
            <AppInput v-model="settings.tv_naming_format" placeholder="{{ '{' }}{{ '{' }}title{{ '}' }}{% if year %} ({{ '{' }}{{ '{' }}year{{ '}' }}){% endif %}/Season {{ '{' }}{{ '{' }}season{{ '}' }}/{{ '{' }}{{ '{' }}title{{ '}' }}…" />
          </template>
        </SettingsRow>
      </SettingsCard>

      <SettingsCard title="媒体信息标签排序" :accent="ORGANIZE_SETTINGS_ACCENT">
        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('align_media_tags')">
          <template #info>
            <div class="settings-row__label">
              <span>强迫症模式</span>
              <SettingsHelpTooltip title="强迫症模式说明">
                <p>开启后，同一后缀文件将保持媒体信息一致。</p>
              </SettingsHelpTooltip>
            </div>
          </template>
          <template #control>
            <SettingsBoolSegment v-model="settings.align_media_tags" label="强迫症模式" />
          </template>
        </SettingsRow>

        <div class="mo-tag-row">
          <SettingsRow
            :show-changed-badge="true"
            :changed="isFieldChanged('media_tag_order')"
          >
            <template #info>
              <div class="settings-row__label">
                <span>媒体信息顺序</span>
                <SettingsHelpTooltip title="媒体信息顺序说明">
                  <p>拖拽标签调整顺序，点击 × 移除。文件名按此顺序生成媒体信息。</p>
                </SettingsHelpTooltip>
              </div>
            </template>
            <template #control>
              <div class="mo-tag-editor">
                <div
                  ref="tagEditorRef"
                  class="mo-tag-editor__active"
                  :class="{ 'mo-tag-editor__active--dragging': draggingTagIndex !== null }"
                >
                  <template v-if="draggingTagIndex === null">
                    <span
                      v-for="(key, index) in tagOrder"
                      :key="key"
                      class="tag-chip"
                      :data-tag-index="index"
                      @pointerdown="startTagPointerDrag(index, $event)"
                    >
                      <span class="tag-chip__text">{{ TAG_LABELS[key] }}</span>
                      <span class="tag-chip__remove" @click.stop="removeTag(key)">×</span>
                    </span>
                    <span v-if="tagOrder.length === 0" class="mo-tag-editor__placeholder">点击下方标签添加</span>
                  </template>
                  <template v-else>
                    <template v-for="slot in tagOrder.length + 1" :key="`tag-slot-${slot - 1}`">
                      <span
                        v-if="insertIndex === slot - 1"
                        class="tag-insert-preview"
                        :data-insert-index="slot - 1"
                      />
                      <span
                        v-if="slot - 1 < tagOrder.length && slot - 1 !== draggingTagIndex"
                        :key="tagOrder[slot - 1]"
                        class="tag-chip"
                        :data-tag-index="slot - 1"
                      >
                        <span class="tag-chip__text">{{ TAG_LABELS[tagOrder[slot - 1]] }}</span>
                        <span class="tag-chip__remove" @click.stop="removeTag(tagOrder[slot - 1])">×</span>
                      </span>
                    </template>
                  </template>
                </div>
                <Teleport to="body">
                  <span
                    v-if="tagGhost.visible"
                    class="tag-chip tag-chip--ghost"
                    :style="tagGhostStyle"
                  >
                    <span class="tag-chip__text">{{ tagGhost.label }}</span>
                    <span class="tag-chip__remove tag-chip__remove--ghost" aria-hidden="true">×</span>
                  </span>
                </Teleport>
                <div v-if="disabledTags.length" class="mo-tag-editor__pool">
                  <span
                    v-for="key in disabledTags"
                    :key="key"
                    class="tag-chip tag-chip--add"
                    @click="addTag(key)"
                  >
                    <span class="tag-chip__addon">+</span>
                    <span class="tag-chip__text">{{ TAG_LABELS[key] }}</span>
                  </span>
                </div>
              </div>
            </template>
          </SettingsRow>
        </div>
      </SettingsCard>

      <SettingsCard title="制作组（MoviePilot）" :accent="ORGANIZE_SETTINGS_ACCENT">
        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('release_groups')">
          <template #info>
            <div class="settings-row__label">
              <span>自定义制作组</span>
              <SettingsHelpTooltip title="制作组说明">
                <p>逗号分隔，追加到内置制作组表（MoviePilot 同款）。用于识别文件名尾部的制作组/字幕组标记。</p>
                <p>例：<code>NHDWEB,UBWEB,FRDS,CHDWEB</code></p>
                <p>识别到后可用于模板变量 <code>&#123;&#123; releaseGroup &#125;&#125;</code>。</p>
              </SettingsHelpTooltip>
            </div>
          </template>
          <template #control>
            <AppInput v-model="settings.release_groups" placeholder="NHDWEB,UBWEB,FRDS,CHDWEB" />
          </template>
        </SettingsRow>
      </SettingsCard>

      <SettingsCard title="分类策略（MoviePilot）" :accent="ORGANIZE_SETTINGS_ACCENT">
        <SettingsRow :show-changed-badge="true" :changed="isFieldChanged('category_map')">
          <template #info>
            <div class="settings-row__label">
              <span>分类映射（JSON）</span>
              <SettingsHelpTooltip title="分类策略说明">
                <p>与 MoviePilot 官方分类策略一致：JSON，movie=电影、tv=电视剧；按顺序匹配首个命中，无条件规则为兜底（不写条件即兜底）。</p>
                <p>条件字段（同 MoviePilot）：<code>genre_ids</code> 类型、<code>original_language</code> 语种、<code>origin_country</code> 地区（剧）、<code>production_countries</code> 地区（电影）、<code>release_year</code> 年份；多值用逗号分隔，支持 <code>!</code> 排除、<code>YYYY-YYYY</code> 范围。</p>
                <p>电影默认：动画电影(16) / 华语电影(zh,cn,bo,za) / 外语电影(兜底)</p>
                <p>电视剧默认：国漫(16+CN,TW,HK) / 日番(16+JP) / 纪录片(99) / 儿童(10762) / 综艺(10764,10767) / 国产剧(CN,TW,HK) / 欧美剧(US,FR,GB,DE,ES,IT,NL,PT,RU,UK) / 日韩剧(JP,KP,KR,TH,IN,SG) / 未分类(兜底)</p>
                <p>整理时命中规则会在作品目录外自动创建分类目录，如 <code>华语电影/流浪地球 (2019)/流浪地球 (2019).mkv</code>。</p>
              </SettingsHelpTooltip>
            </div>
          </template>
          <template #control>
            <AppInput v-model="settings.category_map" placeholder='{ "movie": { "华语电影": { "original_language": "zh" } } }' />
          </template>
        </SettingsRow>
      </SettingsCard>

      <SettingsCard title="整理模板测试（Jinja2）" :accent="ORGANIZE_SETTINGS_ACCENT">
        <template #head-aside>
          <span class="mo-tpl-tip" v-text="'支持 {{{{ title }}}} / {{% if %}} / en_title 等 MoviePilot 兼容变量'"></span>
        </template>
        <SettingsRow>
          <template #info>
            <div class="settings-row__label">
              <span>未整理文件名</span>
              <SettingsHelpTooltip title="未整理文件名说明">
                <p>填一个待整理的文件名，测试时会先解析它，再用上方「命名模板（MoviePilot）」的电影/电视剧模板渲染整理结果。</p>
                <p>留空则使用内置示例（电视剧《沙丘》S01E05）。</p>
              </SettingsHelpTooltip>
            </div>
          </template>
          <template #control>
            <AppInput v-model="tplTest.filename" placeholder="流浪地球.2019.2160p.mkv" />
          </template>
        </SettingsRow>
        <div class="mo-settings__card-head">
          <AppButton type="button" variant="secondary" size="sm" :disabled="tplTesting" @click="testTemplates">
            {{ tplTesting ? "测试中…" : "测试模板" }}
          </AppButton>
        </div>
        <div v-if="tplResult" class="mo-tpl-result">
          <div class="mo-tpl-result__row mo-tpl-result__row--path">
            <span class="mo-tpl-result__label">整理后完整路径</span>
            <code>{{ tplResult.full_path || "（请先在上方填写命名模板）" }}</code>
          </div>
          <div class="mo-tpl-result__row">
            <span class="mo-tpl-result__label">识别类型</span>
            <code>{{ tplResult.is_tv ? "电视剧" : "电影" }}</code>
          </div>
          <div v-if="tplResult.category" class="mo-tpl-result__row">
            <span class="mo-tpl-result__label">分类目录</span>
            <code>{{ tplResult.category }}（按当前分类策略匹配示例）</code>
          </div>
        </div>
      </SettingsCard>
    </template>
  </div>
</template>

<style scoped>
.mo-settings {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.mo-tpl-tip {
  font-size: 12px;
  color: var(--text-muted);
}

.mo-tpl-result {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  border: 1px solid var(--border-soft);
  border-radius: 10px;
  background: var(--surface-sunken);
}

.mo-tpl-result__row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.mo-tpl-result__label {
  flex: 0 0 76px;
  font-size: 12px;
  color: var(--text-muted);
}

.mo-tpl-result__row code {
  font-size: 12px;
  word-break: break-all;
}

.mo-settings__card-head {
  display: flex;
  justify-content: flex-end;
  margin: -4px 0 8px;
}

.mo-tag-row :deep(.settings-row) {
  align-items: flex-start;
}

.mo-tag-row :deep(.settings-row__info) {
  padding-top: 4px;
}

.mo-tag-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.mo-tag-editor__active {
  --tag-chip-width: calc(4em + 40px);
  --tag-chip-height: 32px;
}

.mo-tag-editor__active,
.mo-tag-editor__pool {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.mo-tag-editor__active--dragging {
  flex-wrap: nowrap;
}

:global(body.mo-tag-dragging) {
  cursor: grabbing;
  user-select: none;
}

.mo-tag-editor__placeholder {
  font-size: 13px;
  color: var(--text-muted);
}

.tag-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  box-sizing: border-box;
  flex: 0 0 var(--tag-chip-width, calc(4em + 40px));
  width: var(--tag-chip-width, calc(4em + 40px));
  min-height: var(--tag-chip-height, 32px);
  padding: 6px 10px;
  background: color-mix(in srgb, var(--brand) 10%, var(--surface));
  border: 1px solid color-mix(in srgb, var(--brand) 25%, var(--border));
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.2;
  color: var(--brand);
  cursor: grab;
  user-select: none;
  touch-action: none;
}

.tag-chip:active {
  cursor: grabbing;
}

.tag-insert-preview {
  display: inline-flex;
  box-sizing: border-box;
  flex: 0 0 var(--tag-chip-width, calc(4em + 40px));
  width: var(--tag-chip-width, calc(4em + 40px));
  height: var(--tag-chip-height, 32px);
  min-height: var(--tag-chip-height, 32px);
  border: 1px dashed var(--brand);
  border-radius: 6px;
  background: color-mix(in srgb, var(--brand) 6%, transparent);
}

.tag-chip--ghost {
  position: fixed;
  z-index: 10000;
  margin: 0;
  pointer-events: none;
  cursor: grabbing;
  box-sizing: border-box;
  box-shadow: 0 8px 20px color-mix(in srgb, var(--brand) 22%, transparent);
}

.tag-chip__text {
  flex: 0 0 4em;
  width: 4em;
  text-align: center;
  font-weight: 500;
}

.tag-chip__addon {
  flex: 0 0 12px;
  width: 12px;
  text-align: center;
  font-weight: 500;
}

.tag-chip__remove {
  flex: 0 0 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  font-size: 12px;
  cursor: pointer;
}

.tag-chip__remove:hover {
  background: color-mix(in srgb, var(--brand) 20%, transparent);
}

.tag-chip__remove--ghost {
  visibility: hidden;
  pointer-events: none;
}

.tag-chip--add {
  cursor: pointer;
  flex: 0 0 calc(4em + 28px);
  width: calc(4em + 28px);
  min-height: var(--tag-chip-height, 32px);
  background: var(--surface-sunken);
  border: 1px dashed var(--border-soft);
  color: var(--text-muted);
}

.tag-chip--add .tag-chip__text {
  color: var(--text-muted);
}

.tag-chip--add:hover {
  background: color-mix(in srgb, var(--brand) 8%, var(--surface));
  border-color: color-mix(in srgb, var(--brand) 40%, var(--border));
  color: var(--brand);
}
</style>
