<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Copy, Check } from "lucide-vue-next";
import Dialog from "@/components/ui/Dialog.vue";
import Button from "@/components/ui/Button.vue";
import { generateSnippets, type SnippetReq } from "@/composables/useCodeSnippet";

const props = defineProps<{ open: boolean; req?: SnippetReq | null }>();
defineEmits<{ (e: "close"): void }>();

const { t } = useI18n();
const active = ref(0);
const copied = ref(false);

// 对话框打开时重置到首个语言
watch(
  () => props.open,
  (v) => {
    if (v) {
      active.value = 0;
      copied.value = false;
    }
  }
);

const snippets = computed(() =>
  generateSnippets(
    props.req ?? { method: "GET", url: "", headers: {}, body: "" }
  )
);

async function copy() {
  const code = snippets.value[active.value]?.code;
  if (!code) return;
  try {
    await navigator.clipboard.writeText(code);
    copied.value = true;
    setTimeout(() => (copied.value = false), 1500);
  } catch {
    /* 剪贴板不可用时忽略 */
  }
}
</script>

<template>
  <Dialog :open="open" :title="t('common.codeSnippet')" @close="$emit('close')">
    <div class="flex h-[60vh] flex-col gap-3">
      <!-- 语言 Tab -->
      <div class="flex flex-wrap gap-1.5">
        <button
          v-for="(s, i) in snippets"
          :key="s.lang"
          class="rounded-lg px-3 py-1.5 text-xs font-medium transition-colors"
          :class="
            i === active
              ? 'bg-primary/20 text-primary'
              : 'bg-surface text-muted hover:bg-border/30 hover:text-foreground'
          "
          @click="active = i"
        >
          {{ s.label }}
        </button>
      </div>

      <!-- 代码区 -->
      <div class="relative flex-1 overflow-hidden rounded-xl border border-border bg-[#0B0E14]">
        <Button
          variant="ghost"
          size="sm"
          class="absolute right-2 top-2 z-10"
          @click="copy"
        >
          <Check v-if="copied" :size="14" class="text-emerald-400" />
          <Copy v-else :size="14" />
          {{ copied ? t("common.copied") : t("common.copy") }}
        </Button>
        <pre class="h-full overflow-auto scroll-thin p-4 pt-12 text-[12.5px] leading-relaxed text-emerald-200/90"><code>{{ snippets[active]?.code }}</code></pre>
      </div>
    </div>
  </Dialog>
</template>
