<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter, useRoute } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useToast } from "@/composables/useToast";
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

async function submit() {
  if (!username.value) return toast.error(t("auth.usernameRequired"));
  if (!password.value) return toast.error(t("auth.passwordRequired"));
  loading.value = true;
  try {
    await auth.login(username.value, password.value);
    toast.success(t("auth.loginSuccess"));
    router.push((route.query.redirect as string) || "/projects");
  } catch (e: any) {
    const msg = e?.response?.data?.message;
    toast.error(msg === "invalid username or password" ? t("auth.invalidCred") : t("common.error"));
  } finally {
    loading.value = false;
  }
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

      <form class="space-y-4 rounded-xl border border-border bg-surface p-6" @submit.prevent="submit">
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
