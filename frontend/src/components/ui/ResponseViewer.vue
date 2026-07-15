<script setup lang="ts">
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { Copy, FileJson, FileText, Eye } from "lucide-vue-next";
import { useToast } from "@/composables/useToast";

const props = defineProps<{
  body: string;
  contentType?: string;
}>();

const { t } = useI18n();
const toast = useToast();
const tab = ref<"pretty" | "raw" | "preview">("pretty");

const isJson = computed(() => /json/i.test(props.contentType || "") || looksJson(props.body));
const isHtml = computed(() => /html/i.test(props.contentType || ""));

function looksJson(s: string) {
  const x = s.trim();
  return x.startsWith("{") || x.startsWith("[");
}

const pretty = computed(() => {
  if (isJson.value) {
    try {
      return JSON.stringify(JSON.parse(props.body), null, 2);
    } catch {
      return props.body;
    }
  }
  return props.body;
});

function escapeHtml(s: string) {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

const highlighted = computed(() => {
  const src = isJson.value ? pretty.value : props.body;
  if (!isJson.value) return escapeHtml(src);
  return escapeHtml(src).replace(
    /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+-]?\d+)?)/g,
    (m) => {
      let cls = "text-orange-400";
      if (/^"/.test(m)) cls = /:$/.test(m) ? "text-sky-400" : "text-emerald-400";
      else if (/true|false|null/.test(m)) cls = "text-purple-400";
      return `<span class="${cls}">${m}</span>`;
    },
  );
});

function copy() {
  navigator.clipboard.writeText(props.body);
  toast.success(t("common.copied"));
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex items-center gap-1 border-b border-border px-2 py-1.5">
      <button
        class="flex items-center gap-1 rounded-md px-2 py-1 text-xs transition-colors"
        :class="tab === 'pretty' ? 'bg-primary/15 text-primary' : 'text-muted hover:text-foreground'"
        @click="tab = 'pretty'"
      >
        <FileJson :size="13" /> Pretty
      </button>
      <button
        class="flex items-center gap-1 rounded-md px-2 py-1 text-xs transition-colors"
        :class="tab === 'raw' ? 'bg-primary/15 text-primary' : 'text-muted hover:text-foreground'"
        @click="tab = 'raw'"
      >
        <FileText :size="13" /> Raw
      </button>
      <button
        class="flex items-center gap-1 rounded-md px-2 py-1 text-xs transition-colors disabled:cursor-not-allowed disabled:opacity-40"
        :class="tab === 'preview' ? 'bg-primary/15 text-primary' : 'text-muted hover:text-foreground'"
        :disabled="!isHtml"
        @click="tab = 'preview'"
      >
        <Eye :size="13" /> Preview
      </button>
      <span class="flex-1" />
      <button class="rounded-md p-1 text-muted transition-colors hover:text-foreground" title="Copy" @click="copy">
        <Copy :size="14" />
      </button>
    </div>

    <div class="flex-1 overflow-auto scroll-thin bg-surface p-3">
      <pre
        v-show="tab === 'pretty'"
        class="whitespace-pre-wrap break-all font-mono text-xs leading-relaxed text-foreground"
        v-html="highlighted"
      />
      <pre
        v-show="tab === 'raw'"
        class="whitespace-pre-wrap break-all font-mono text-xs text-foreground"
      >{{ body }}</pre>
      <iframe
        v-show="tab === 'preview' && isHtml"
        :srcdoc="body"
        sandbox="allow-same-origin"
        class="h-full min-h-[240px] w-full border-0 bg-white"
      />
      <div v-if="tab === 'preview' && !isHtml" class="text-xs text-muted">
        Preview 仅支持 HTML 响应
      </div>
    </div>
  </div>
</template>
