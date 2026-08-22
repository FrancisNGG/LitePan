<script setup lang="ts">
import { onMounted, ref } from "vue";
import { getApiErrorMessage } from "@/api/client";
import {
  fetchMediaEnhanceStatus,
  fetchMediaOrganizeSettings,
  saveMediaOrganizeSettings,
  setMediaEnhanceEnabled,
  testMediaOrganizeTemplates,
} from "@/api/mediaOrganize";
import AppButton from "@/components/base/AppButton.vue";
import AppInput from "@/components/base/AppInput.vue";
import AppModal from "@/components/base/AppModal.vue";
import CloudToolCard from "@/components/admin/CloudToolCard.vue";
import SettingsCard from "@/components/admin/SettingsCard.vue";
import SettingsHelpTooltip from "@/components/admin/SettingsHelpTooltip.vue";
import SettingsRow from "@/components/admin/SettingsRow.vue";
import { toast } from "@/composables/useToast";

const props = withDefaults(defineProps<{ searchQuery?: string }>(), { searchQuery: "" });

const status = ref({ enabled: false, available: false });
const saving = ref(false);
const configOpen = ref(false);
const configLoading = ref(false);
const configSaving = ref(false);
const tplTesting = ref(false);
const tplResult = ref<{ full_path?: string; is_tv?: boolean; category?: string } | null>(null);
const tplTestFile = ref("流浪地球.2019.2160p.mkv");

const settings = ref<{
  movie_naming_format: string;
  tv_naming_format: string;
  category_map: string;
  release_groups: string;
}>({
  movie_naming_format: "",
  tv_naming_format: "",
  category_map: "",
  release_groups: "",
});

function matches(title: string) {
  const q = props.searchQuery.trim().toLowerCase();
  return !q || title.toLowerCase().includes(q);
}

async function load() {
  status.value = await fetchMediaEnhanceStatus().catch(() => ({ enabled: false, available: false }));
}

onMounted(() => {
  void load();
});

async function toggleEnabled() {
  saving.value = true;
  const next = !status.value.enabled;
  try {
    await setMediaEnhanceEnabled(next);
    status.value.enabled = next;
    toast.success(next ? "已启用媒体整理增强" : "已停用媒体整理增强（恢复官方行为）");
  } catch (e) {
    toast.error(getApiErrorMessage(e, "切换开关失败"));
  } finally {
    saving.value = false;
  }
}

async function openConfig() {
  configOpen.value = true;
  configLoading.value = true;
  tplResult.value = null;
  try {
    const data = await fetchMediaOrganizeSettings();
    settings.value = {
      movie_naming_format: data.movie_naming_format ?? "",
      tv_naming_format: data.tv_naming_format ?? "",
      category_map: data.category_map ?? "",
      release_groups: data.release_groups ?? "",
    };
  } catch (e) {
    toast.error(getApiErrorMessage(e, "加载配置失败"));
  } finally {
    configLoading.value = false;
  }
}

function closeConfig() {
  if (configSaving.value) return;
  configOpen.value = false;
}

async function saveConfig() {
  configSaving.value = true;
  try {
    const saved = await saveMediaOrganizeSettings({
      movie_naming_format: settings.value.movie_naming_format,
      tv_naming_format: settings.value.tv_naming_format,
      category_map: settings.value.category_map,
      release_groups: settings.value.release_groups,
    });
    settings.value = {
      movie_naming_format: saved.movie_naming_format ?? "",
      tv_naming_format: saved.tv_naming_format ?? "",
      category_map: saved.category_map ?? "",
      release_groups: saved.release_groups ?? "",
    };
    toast.success("媒体整理增强配置已保存");
  } catch (e) {
    toast.error(getApiErrorMessage(e, "保存配置失败"));
  } finally {
    configSaving.value = false;
  }
}

async function testTemplates() {
  tplTesting.value = true;
  tplResult.value = null;
  try {
    tplResult.value = await testMediaOrganizeTemplates({
      movie_naming_template: settings.value.movie_naming_format,
      tv_naming_template: settings.value.tv_naming_format,
      filename: tplTestFile.value,
    });
  } catch (e) {
    toast.error(getApiErrorMessage(e, "模板测试失败"));
  } finally {
    tplTesting.value = false;
  }
}
</script>

<template>
  <div v-show="matches('媒体整理增强')">
    <CloudToolCard
      :enabled="status.enabled"
      name="媒体整理增强"
      driver="作用于媒体整理任务 · 命名模板 + 自动分类"
      logo-src="/logos/media-organize.svg"
      logo-alt="媒体整理增强"
      :tags="[
        { label: '参考 MoviePilot 整理模块' },
        { label: '实验性', variant: 'warn' },
      ]"
    >
      <template #toggle>
        <button
          class="check-toggle"
          type="button"
          :class="{ on: status.enabled }"
          :aria-label="status.enabled ? '停用媒体整理增强' : '启用媒体整理增强'"
          :disabled="saving || !status.available"
          title="启用 / 停用"
          @click="toggleEnabled"
        >
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path
              d="M3.5 8.5 6.5 11.5 12.5 4.5"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </template>
      开启后，媒体整理任务可使用 Jinja2 命名模板（渲染完整路径，自动拆分作品目录/季目录/文件名）与 TMDB 自动分类。
      关闭时媒体整理保持官方默认行为。模板语法参考 MoviePilot 整理模块。
      <template #actions>
        <AppButton variant="secondary" :disabled="configSaving" @click="openConfig">
          配置与测试
        </AppButton>
      </template>
    </CloudToolCard>

    <AppModal :open="configOpen" title="媒体整理增强 · 配置" size="lg" @close="closeConfig">
      <div v-if="configLoading" class="me-loading">加载中…</div>
      <template v-else>
        <div class="me-config">
          <SettingsCard title="命名模板">
            <template #head-aside>
              <span class="me-tip">Jinja2 完整路径模板，按 / 自动拆分为作品目录/季目录/文件名</span>
            </template>
            <SettingsRow>
              <template #info>
                <div class="settings-row__label">
                  <span>电影命名模板</span>
                  <SettingsHelpTooltip title="电影命名模板说明">
                    <p>Jinja2 模板，渲染<code>完整路径</code>（目录/文件名）。</p>
                    <p>示例：<code>{{ '{' }}{{ '{' }}title{{ '}' }}{% if year %} ({{ '{' }}{{ '{' }}year{{ '}' }}){% endif %}/{{ '{' }}{{ '{' }}title{{ '}' }} - {{ '{' }}{{ '{' }}fileExt{{ '}' }}{{ '}' }}</code></p>
                    <p>留空则使用任务级模板。</p>
                  </SettingsHelpTooltip>
                </div>
              </template>
              <template #control>
                <AppInput v-model="settings.movie_naming_format" placeholder="{{ '{' }}{{ '{' }}title{{ '}' }}{% if year %} ({{ '{' }}{{ '{' }}year{{ '}' }}){% endif %}/{{ '{' }}{{ '{' }}title{{ '}' }}…" />
              </template>
            </SettingsRow>
            <SettingsRow>
              <template #info>
                <div class="settings-row__label">
                  <span>电视剧命名模板</span>
                  <SettingsHelpTooltip title="电视剧命名模板说明">
                    <p>Jinja2 模板，渲染<code>完整路径</code>（作品目录/季目录/文件名）。</p>
                    <p>留空则使用任务级模板。</p>
                  </SettingsHelpTooltip>
                </div>
              </template>
              <template #control>
                <AppInput v-model="settings.tv_naming_format" placeholder="{{ '{' }}{{ '{' }}title{{ '}' }}{% if year %} ({{ '{' }}{{ '{' }}year{{ '}' }}){% endif %}/Season {{ '{' }}{{ '{' }}season{{ '}' }}/{{ '{' }}{{ '{' }}title{{ '}' }}…" />
              </template>
            </SettingsRow>
          </SettingsCard>

          <SettingsCard title="制作组">
            <SettingsRow>
              <template #info>
                <div class="settings-row__label">
                  <span>自定义制作组</span>
                  <SettingsHelpTooltip title="制作组说明">
                    <p>逗号分隔，追加到内置制作组表。用于识别文件名尾部的制作组/字幕组标记。</p>
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

          <SettingsCard title="分类策略">
            <SettingsRow>
              <template #info>
                <div class="settings-row__label">
                  <span>分类映射（JSON）</span>
                  <SettingsHelpTooltip title="分类策略说明">
                    <p>JSON：movie=电影、tv=电视剧；按顺序匹配首个命中，无条件规则为兜底。</p>
                    <p>条件字段：<code>genre_ids</code> 类型、<code>original_language</code> 语种、<code>origin_country</code> 地区（剧）、<code>production_countries</code> 地区（电影）、<code>release_year</code> 年份；多值逗号分隔，支持 <code>!</code> 排除、<code>YYYY-YYYY</code> 范围。</p>
                    <p>整理时命中规则会在作品目录外自动创建分类目录，如 <code>华语电影/流浪地球 (2019)/流浪地球 (2019).mkv</code>。</p>
                  </SettingsHelpTooltip>
                </div>
              </template>
              <template #control>
                <AppInput v-model="settings.category_map" placeholder='{ "movie": { "华语电影": { "original_language": "zh" } } }' />
              </template>
            </SettingsRow>
          </SettingsCard>

          <SettingsCard title="模板测试（Jinja2）">
            <SettingsRow>
              <template #info>
                <div class="settings-row__label">
                  <span>未整理文件名</span>
                  <SettingsHelpTooltip title="未整理文件名说明">
                    <p>填一个待整理的文件名，测试时会先解析它，再用上方电影/电视剧模板渲染整理结果。</p>
                    <p>留空则使用内置示例。</p>
                  </SettingsHelpTooltip>
                </div>
              </template>
              <template #control>
                <AppInput v-model="tplTestFile" placeholder="流浪地球.2019.2160p.mkv" />
              </template>
            </SettingsRow>
            <div class="me-test-actions">
              <AppButton type="button" variant="secondary" size="sm" :disabled="tplTesting" @click="testTemplates">
                {{ tplTesting ? "测试中…" : "测试模板" }}
              </AppButton>
            </div>
            <div v-if="tplResult" class="me-result">
              <div class="me-result__row me-result__row--path">
                <span class="me-result__label">整理后完整路径</span>
                <code>{{ tplResult.full_path || "（请先在上方填写命名模板）" }}</code>
              </div>
              <div class="me-result__row">
                <span class="me-result__label">识别类型</span>
                <code>{{ tplResult.is_tv ? "电视剧" : "电影" }}</code>
              </div>
              <div v-if="tplResult.category" class="me-result__row">
                <span class="me-result__label">分类目录</span>
                <code>{{ tplResult.category }}（按当前分类策略匹配示例）</code>
              </div>
            </div>
          </SettingsCard>
        </div>

        <div class="me-footer">
          <AppButton type="button" variant="secondary" :disabled="configSaving" @click="closeConfig">
            取消
          </AppButton>
          <AppButton type="button" variant="primary" :disabled="configSaving" @click="saveConfig">
            {{ configSaving ? "保存中…" : "保存配置" }}
          </AppButton>
        </div>
      </template>
    </AppModal>
  </div>
</template>

<style scoped>
.me-config {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.check-toggle {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 0;
  padding: 0;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  background: var(--border);
  color: var(--text-muted);
  transition: background 0.18s ease, color 0.18s ease, box-shadow 0.18s ease;
}
.check-toggle svg {
  width: 14px;
  height: 14px;
}
.check-toggle:hover {
  background: var(--surface-hover);
}
.check-toggle.on {
  background: var(--success);
  color: #fff;
  box-shadow: 0 0 0 4px rgba(16, 185, 129, 0.16);
}
.check-toggle.on:hover {
  background: color-mix(in srgb, var(--success) 88%, #000);
}
.check-toggle:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.me-tip {
  font-size: 12px;
  color: var(--text-muted, #94a3b8);
}
.me-loading {
  padding: 32px 0;
  text-align: center;
  color: var(--text-muted, #94a3b8);
}
.me-test-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}
.me-result {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.me-result__row {
  display: flex;
  gap: 8px;
  align-items: baseline;
  font-size: 13px;
}
.me-result__row--path {
  flex-direction: column;
  gap: 4px;
}
.me-result__label {
  color: var(--text-muted, #94a3b8);
  flex-shrink: 0;
}
.me-result code {
  background: var(--surface-raised, rgba(148, 163, 184, 0.12));
  padding: 2px 6px;
  border-radius: 6px;
  word-break: break-all;
}
.me-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}
</style>
