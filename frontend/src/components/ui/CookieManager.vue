<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Cookie, Copy } from "lucide-vue-next";
import { useToast } from "@/composables/useToast";
import type { RespCookie } from "@/types/protocol";

const props = defineProps<{ cookies: RespCookie[] }>();
const { t } = useI18n();
const toast = useToast();

const headerValue = computed(() => props.cookies.map((c) => `${c.name}=${c.value}`).join("; "));

function copyHeader() {
  navigator.clipboard.writeText(headerValue.value);
  toast.success(t("common.copied"));
}
</script>

<template>
  <div class="space-y-2">
    <div class="flex items-center gap-1.5 text-xs font-medium text-muted">
      <Cookie :size="13" /> {{ t("http.cookies") }}
    </div>
    <div v-if="cookies.length === 0" class="text-xs text-muted/60">无 Set-Cookie</div>
    <div
      v-for="c in cookies"
      :key="(c.domain || '') + c.name"
      class="rounded-lg border border-border bg-surface p-2 text-xs"
    >
      <div class="flex items-center gap-2">
        <span class="font-medium text-foreground">{{ c.name }}</span>
        <span class="text-muted">=</span>
        <span class="break-all text-emerald-400">{{ c.value }}</span>
      </div>
      <div class="mt-1 flex flex-wrap gap-2 text-[11px] text-muted/70">
        <span v-if="c.domain">domain: {{ c.domain }}</span>
        <span v-if="c.path">path: {{ c.path }}</span>
        <span v-if="c.expires">expires: {{ c.expires }}</span>
        <span v-if="c.sameSite">samesite: {{ c.sameSite }}</span>
        <span v-if="c.httpOnly" class="text-amber-400">HttpOnly</span>
        <span v-if="c.secure" class="text-sky-400">Secure</span>
      </div>
    </div>
    <button
      v-if="cookies.length"
      class="flex items-center gap-1 rounded-md px-2 py-1 text-xs text-muted transition-colors hover:bg-border/30 hover:text-foreground"
      @click="copyHeader"
    >
      <Copy :size="13" /> 复制为 Cookie 请求头
    </button>
  </div>
</template>
