<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useThemeStore } from "@/stores/theme";
import { i18n, setLocale } from "@/locales";
import { LogOut, Globe, Terminal, Sun, Moon, User as UserIcon } from "lucide-vue-next";
import Button from "@/components/ui/Button.vue";

const { t } = useI18n();
const router = useRouter();
const auth = useAuthStore();
const theme = useThemeStore();
const menuOpen = ref(false);

function changeLocale(e: Event) {
  setLocale((e.target as HTMLSelectElement).value);
}
function logout() {
  auth.logout();
  router.push("/login");
}
</script>

<template>
  <header class="fixed top-0 inset-x-0 z-40 h-14 flex items-center justify-between border-b border-border glass px-4">
    <div class="flex items-center gap-2">
      <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-primary to-primary-3 text-white shadow-glow">
        <Terminal :size="18" />
      </div>
      <span class="text-lg font-semibold gradient-text">{{ t("common.appName") }}</span>
    </div>

    <div class="flex items-center gap-3">
      <button
        :title="t('common.themeToggle')"
        :aria-label="t('common.themeToggle')"
        class="flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-surface text-foreground hover:border-primary/60"
        @click="theme.toggle()"
      >
        <Sun v-if="theme.theme === 'light'" :size="16" />
        <Moon v-else :size="16" />
      </button>

      <div class="flex items-center gap-1.5 text-muted">
        <Globe :size="16" />
        <select
          :value="i18n.global.locale.value"
          class="h-8 rounded-lg border border-border bg-surface px-2 text-sm text-foreground outline-none focus:border-primary/60"
          @change="changeLocale"
        >
          <option value="zh-CN">中文</option>
          <option value="en-US">English</option>
        </select>
      </div>

      <div class="relative">
        <button
          class="flex h-9 items-center gap-2 rounded-lg border border-border bg-surface px-3 text-sm hover:border-primary/60"
          @click="menuOpen = !menuOpen"
        >
          <span class="flex h-6 w-6 items-center justify-center rounded-full bg-primary/20 text-primary">
            {{ auth.user?.username?.charAt(0)?.toUpperCase() }}
          </span>
          <span class="text-foreground">{{ auth.user?.username }}</span>
        </button>
        <div
          v-if="menuOpen"
          class="absolute right-0 mt-2 w-40 rounded-lg border border-border glass p-1 shadow-glow animate-fade-in-up"
          @mouseleave="menuOpen = false"
        >
          <button
            class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-muted hover:bg-border/30 hover:text-foreground"
            @click="router.push('/account'); menuOpen = false"
          >
            <UserIcon :size="15" /> {{ t("account.title") }}
          </button>
          <button
            class="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-muted hover:bg-border/30 hover:text-foreground"
            @click="logout"
          >
            <LogOut :size="15" /> {{ t("common.logout") }}
          </button>
        </div>
      </div>
    </div>
  </header>
</template>
