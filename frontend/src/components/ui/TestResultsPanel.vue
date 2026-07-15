<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { Assertion } from "@/composables/useScriptRunner";
import Badge from "@/components/ui/Badge.vue";
import { Check, X, Variable, Terminal } from "lucide-vue-next";

const props = defineProps<{
  assertions: Assertion[];
  extracted: { varName: string; value: string }[];
  logs: string[];
  error?: string;
}>();

const { t } = useI18n();

const passCount = computed(() => props.assertions.filter((a) => a.passed).length);
const failCount = computed(() => props.assertions.length - passCount.value);
const hasAny = computed(
  () =>
    props.assertions.length > 0 ||
    props.extracted.length > 0 ||
    props.logs.length > 0 ||
    !!props.error
);
</script>

<template>
  <div v-if="hasAny" class="space-y-3 text-xs">
    <!-- 脚本错误 -->
    <div
      v-if="error"
      class="rounded-md border border-danger/40 bg-danger/10 p-2 text-danger"
    >
      ⚠ {{ t("common.script.execError") }}：{{ error }}
    </div>

    <!-- 断言结果 -->
    <div v-if="assertions.length">
      <div class="mb-1 flex items-center gap-2 font-medium text-muted">
        {{ t("common.script.tests") }}
        <Badge :tone="failCount ? 'danger' : 'success'">
          {{ passCount }}/{{ assertions.length }}
        </Badge>
      </div>
      <div class="space-y-1">
        <div
          v-for="(a, i) in assertions"
          :key="i"
          class="flex items-start gap-2 rounded bg-surface/60 px-2 py-1"
        >
          <component
            :is="a.passed ? Check : X"
            :size="13"
            class="mt-0.5 shrink-0"
            :class="a.passed ? 'text-emerald-400' : 'text-danger'"
          />
          <div>
            <div :class="a.passed ? 'text-foreground' : 'text-danger'">{{ a.name }}</div>
            <div v-if="!a.passed && a.message" class="text-danger/80">{{ a.message }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 提取结果 -->
    <div v-if="extracted.length">
      <div class="mb-1 flex items-center gap-2 font-medium text-muted">
        <Variable :size="13" /> {{ t("common.script.extracted") }}
      </div>
      <div class="space-y-1">
        <div
          v-for="(e, i) in extracted"
          :key="i"
          class="flex items-center gap-2 rounded bg-surface/60 px-2 py-1"
        >
          <code class="text-primary">{{ e.varName }}</code>
          <span class="text-muted">=</span>
          <code class="truncate text-foreground">{{ e.value }}</code>
        </div>
      </div>
    </div>

    <!-- 控制台日志 -->
    <div v-if="logs.length">
      <div class="mb-1 flex items-center gap-2 font-medium text-muted">
        <Terminal :size="13" /> {{ t("common.script.console") }}
      </div>
      <pre class="max-h-40 overflow-auto scroll-thin rounded bg-black/40 p-2 text-[11px] leading-relaxed text-emerald-200/90">{{ logs.join("\n") }}</pre>
    </div>
  </div>
</template>
