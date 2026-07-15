<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { Construction } from "lucide-vue-next";
import Card from "@/components/ui/Card.vue";

defineProps<{ name: string }>();
const { t } = useI18n();
</script>

<template>
  <div class="flex h-full items-center justify-center p-6">
    <Card class="flex max-w-md flex-col items-center gap-3 p-8 text-center">
      <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-primary/30 to-primary-3/20 text-primary">
        <Construction :size="24" />
      </div>
      <h3 class="text-lg font-semibold text-foreground">{{ name }}</h3>
      <p class="text-sm text-muted">
        {{ t("common.comingSoon") || "MCP、AI 接口调试计划后续完成" }}
      </p>
    </Card>
  </div>
</template>
