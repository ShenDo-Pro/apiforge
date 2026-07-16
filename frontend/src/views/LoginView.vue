<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter, useRoute } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useToast } from "@/composables/useToast";
import { resetPassword } from "@/api/auth";
import Button from "@/components/ui/Button.vue";
import Input from "@/components/ui/Input.vue";
import { Terminal } from "lucide-vue-next";

const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const auth = useAuthStore();
const toast = useToast();

const username = ref("");
const password = ref("");
const loading = ref(false);

// 首次登录强制改密（H6）：登录成功后若后端标记 needReset，弹出改密框，
// 改密完成才放行进入系统。
const forcingReset = ref(false);
const oldPwd = ref("");
const newPwd = ref("");
const resetLoading = ref(false);

async function submit() {
  if (!username.value) return toast.error(t("auth.usernameRequired"));
  if (!password.value) return toast.error(t("auth.passwordRequired"));
  loading.value = true;
  try {
    await auth.login(username.value, password.value);
    if (auth.user?.needReset) {
      forcingReset.value = true;
      oldPwd.value = password.value; // 首次改密时旧口令即本次登录口令
      return;
    }
    enter();
  } catch (e: any) {
    const msg = e?.response?.data?.message;
    toast.error(msg === "invalid username or password" ? t("auth.invalidCred") : t("common.error"));
  } finally {
    loading.value = false;
  }
}

async function doReset() {
  if (newPwd.value.length < 8) return toast.error(t("auth.pwdTooShort"));
  resetLoading.value = true;
  try {
    await resetPassword(oldPwd.value, newPwd.value);
    toast.success(t("auth.pwdChanged"));
    enter();
  } catch (e: any) {
    toast.error(e?.response?.data?.message || t("common.error"));
  } finally {
    resetLoading.value = false;
  }
}

function enter() {
  router.push((route.query.redirect as string) || "/projects");
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background px-4">
    <div class="w-full max-w-sm">
      <div class="mb-7 flex flex-col items-center text-center">
        <div class="mb-3 flex h-11 w-11 items-center justify-center rounded-xl bg-primary/15 text-primary">
          <Terminal :size="22" />
        </div>
        <h1 class="text-xl font-semibold text-foreground">{{ t("common.appName") }}</h1>
        <p class="mt-1 text-sm text-muted">{{ t("auth.subtitle") }}</p>
      </div>

      <!-- 首次强制改密 -->
      <form v-if="forcingReset" class="space-y-4 rounded-xl border border-border bg-surface p-6" @submit.prevent="doReset">
        <p class="text-sm text-foreground">{{ t("auth.forceResetHint") }}</p>
        <div>
          <label class="mb-1 block text-xs font-medium text-muted">{{ t("auth.newPassword") }}</label>
          <Input v-model="newPwd" type="password" autocomplete="new-password" :placeholder="t('auth.newPassword')" />
        </div>
        <Button type="submit" class="w-full" :disabled="resetLoading">
          {{ resetLoading ? t("auth.changing") : t("auth.confirmChange") }}
        </Button>
      </form>

      <!-- 普通登录 -->
      <form v-else class="space-y-4 rounded-xl border border-border bg-surface p-6" @submit.prevent="submit">
        <div>
          <label class="mb-1 block text-xs font-medium text-muted">{{ t("auth.username") }}</label>
          <Input v-model="username" autocomplete="username" :placeholder="t('auth.username')" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-muted">{{ t("auth.password") }}</label>
          <Input v-model="password" type="password" autocomplete="current-password" :placeholder="t('auth.password')" />
        </div>
        <Button type="submit" class="w-full" :disabled="loading">
          {{ loading ? t("auth.loggingIn") : t("auth.loginBtn") }}
        </Button>
      </form>

      <p class="mt-4 text-center text-xs text-muted/70">
        {{ t("auth.defaultHint") }}
      </p>
    </div>
  </div>
</template>
