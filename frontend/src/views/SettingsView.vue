<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { useThemeStore } from "@/stores/theme";
import { i18n, setLocale } from "@/locales";
import { useProjectStore } from "@/stores/project";
import { Sun, Moon, Languages } from "lucide-vue-next";

const props = defineProps<{ projectId: number }>();
const { t } = useI18n();
const theme = useThemeStore();
const project = useProjectStore();

function changeLocale(e: Event) {
  setLocale((e.target as HTMLSelectElement).value);
}
</script>

<template>
  <div class="h-full overflow-y-auto scroll-thin">
    <div class="mx-auto max-w-3xl space-y-6 p-6">
      <h2 class="text-lg font-semibold text-foreground">{{ t("common.navSettings") }}</h2>

      <!-- 项目信息 -->
      <div class="rounded-xl border border-border bg-surface p-4">
        <div class="mb-3 text-sm font-medium text-muted">{{ t("common.appName") }}</div>
        <div class="space-y-1 text-sm">
          <div class="flex justify-between"><span class="text-muted">{{ t("common.name") }}</span><span class="text-foreground">{{ project.current?.name }}</span></div>
          <div class="flex justify-between"><span class="text-muted">{{ t("common.description") }}</span><span class="text-foreground">{{ project.current?.description || "—" }}</span></div>
        </div>
      </div>

      <!-- 偏好 -->
      <div class="space-y-3 rounded-xl border border-border bg-surface p-4">
        <div class="text-sm font-medium text-muted">{{ t("common.preferences") }}</div>

        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2 text-sm text-foreground">
            <component :is="theme.theme === 'light' ? Sun : Moon" :size="16" />
            {{ t("common.themeToggle") }}
          </div>
          <button
            class="flex h-9 items-center gap-2 rounded-lg border border-border bg-background px-3 text-sm hover:border-primary/60"
            @click="theme.toggle()"
          >
            <component :is="theme.theme === 'light' ? Sun : Moon" :size="15" />
            {{ theme.theme === "light" ? t("common.themeLight") : t("common.themeDark") }}
          </button>
        </div>

        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2 text-sm text-foreground">
            <Languages :size="16" />
            {{ t("common.language") }}
          </div>
          <select
            :value="i18n.global.locale.value"
            class="h-9 rounded-lg border border-border bg-background px-2 text-sm text-foreground outline-none focus:border-primary/60"
            @change="changeLocale"
          >
            <option value="zh-CN">中文</option>
            <option value="en-US">English</option>
          </select>
        </div>
      </div>
    </div>
  </div>
</template>
