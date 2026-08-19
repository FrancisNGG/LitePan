<script setup lang="ts">
import { onMounted, ref } from "vue";
import { getApiErrorMessage } from "@/api/client";
import { deleteStrmFile, fetchStrmFiles, type StrmFileEntry } from "@/api/strm";
import AppButton from "@/components/base/AppButton.vue";
import SettingsCard from "@/components/admin/SettingsCard.vue";
import { confirm } from "@/composables/useConfirm";
import { toast } from "@/composables/useToast";
import "@/styles/admin-shared.css";

const loading = ref(false);
const entries = ref<StrmFileEntry[]>([]);
const currentDir = ref("");
const pathStack = ref<string[]>([]);
const error = ref("");

async function load(dir = currentDir.value) {
  loading.value = true;
  error.value = "";
  try {
    entries.value = await fetchStrmFiles({ dir: dir || undefined, recurse: false });
  } catch (e) {
    error.value = getApiErrorMessage(e, "加载 STRM 文件失败");
    entries.value = [];
  } finally {
    loading.value = false;
  }
}

function openDir(entry: StrmFileEntry) {
  if (!entry.is_dir) return;
  pathStack.value.push(currentDir.value);
  currentDir.value = entry.path;
  void load(entry.path);
}

function goUp() {
  const parent = pathStack.value.pop();
  if (parent === undefined) return;
  currentDir.value = parent;
  void load(parent);
}

function goRoot() {
  pathStack.value = [];
  currentDir.value = "";
  void load("");
}

async function remove(entry: StrmFileEntry) {
  const ok = await confirm({
    title: "删除 STRM 文件",
    message: `确定删除「${entry.path}」？${
      entry.is_dir ? "目录及其全部内容将被删除。" : ""
    }`,
    confirmText: "删除",
    danger: true,
  });
  if (!ok) return;
  try {
    await deleteStrmFile(entry.path);
    toast.success("已删除");
    void load();
  } catch (e) {
    toast.error(getApiErrorMessage(e, "删除失败"));
  }
}

function fmtSize(n: number) {
  if (!n) return "-";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

onMounted(() => void load(""));
</script>

<template>
  <SettingsCard title="STRM 文件管理" :accent="'#7c3aed'">
    <p class="settings-help">
      浏览 STRM 输出目录（配置项 strm_dir）下的文件，可进入子目录或删除文件。
    </p>

    <div class="strm-files-toolbar">
      <AppButton size="sm" variant="ghost" @click="goRoot" :disabled="!currentDir">
        根目录
      </AppButton>
      <AppButton size="sm" variant="ghost" @click="goUp" :disabled="!pathStack.length">
        上级目录
      </AppButton>
      <span class="strm-files-path">/{{ currentDir }}</span>
      <AppButton size="sm" variant="ghost" @click="load()" :loading="loading">
        刷新
      </AppButton>
    </div>

    <div v-if="error" class="strm-files-error">{{ error }}</div>

    <div v-if="loading" class="strm-files-empty">加载中…</div>
    <div v-else-if="!entries.length" class="strm-files-empty">
      目录为空{{ currentDir ? "（或 STRM 目录未配置/不存在）" : "" }}
    </div>

    <table v-else class="strm-files-table">
      <thead>
        <tr>
          <th>名称</th>
          <th>大小</th>
          <th>修改时间</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="entry in entries" :key="entry.path">
          <td>
            <span
              class="strm-files-name"
              :class="{ 'is-dir': entry.is_dir }"
              @click="openDir(entry)"
            >
              {{ entry.is_dir ? "📁 " : "📄 " }}{{ entry.name }}
            </span>
          </td>
          <td class="strm-files-size">{{ entry.is_dir ? "-" : fmtSize(entry.size) }}</td>
          <td class="strm-files-time">{{ entry.mod_time }}</td>
          <td class="strm-files-actions">
            <AppButton size="sm" variant="danger" @click="remove(entry)">删除</AppButton>
          </td>
        </tr>
      </tbody>
    </table>
  </SettingsCard>
</template>

<style scoped>
.strm-files-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.strm-files-path {
  font-family: var(--font-mono, monospace);
  font-size: 12.5px;
  color: var(--text-secondary, #666);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.strm-files-error {
  color: #c6432e;
  font-size: 13px;
  margin-bottom: 10px;
}
.strm-files-empty {
  color: var(--text-faint, #999);
  font-size: 13px;
  padding: 18px 0;
  text-align: center;
}
.strm-files-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.strm-files-table th {
  text-align: left;
  padding: 6px 8px;
  border-bottom: 1px solid var(--border, #e5e5e5);
  color: var(--text-faint, #999);
  font-weight: 500;
}
.strm-files-table td {
  padding: 6px 8px;
  border-bottom: 1px solid var(--border-light, #f0f0f0);
}
.strm-files-name {
  cursor: pointer;
}
.strm-files-name.is-dir {
  font-weight: 600;
}
.strm-files-size,
.strm-files-time {
  color: var(--text-secondary, #666);
  white-space: nowrap;
}
.strm-files-actions {
  text-align: right;
  white-space: nowrap;
}
</style>
