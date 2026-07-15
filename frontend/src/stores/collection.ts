import { defineStore } from "pinia";
import {
  listCollections,
  listRequests,
  createCollection,
  updateCollection,
  deleteCollection,
} from "@/api/collection";
import { listHistory } from "@/api/request";
import type { Collection, SavedRequest, RequestHistory } from "@/types/project";

// 集合树与请求/历史缓存，按 id 建立索引便于前端组装树。
export const useCollectionStore = defineStore("collection", {
  state: () => ({
    collections: [] as Collection[],
    // 每个集合下的保存请求缓存，按 collectionId 索引，便于树形按需刷新
    requestsByCollection: {} as Record<number, SavedRequest[]>,
    histories: {} as Record<number, RequestHistory[]>,
  }),
  actions: {
    async fetchCollections(projectId: number) {
      this.collections = await listCollections(projectId);
    },
    async fetchRequests(projectId: number, collectionId: number) {
      this.requestsByCollection[collectionId] = await listRequests(
        projectId,
        collectionId,
      );
      return this.requestsByCollection[collectionId];
    },
    async refreshRequests(projectId: number, collectionId: number) {
      return this.fetchRequests(projectId, collectionId);
    },
    async createCollection(
      projectId: number,
      payload: { parentId?: number | null; name: string },
    ) {
      const c = await createCollection(projectId, payload);
      await this.fetchCollections(projectId);
      return c;
    },
    async renameCollection(
      projectId: number,
      collectionId: number,
      name: string,
    ) {
      await updateCollection(projectId, collectionId, { name });
      await this.fetchCollections(projectId);
    },
    async removeCollection(projectId: number, collectionId: number) {
      await deleteCollection(projectId, collectionId);
      await this.fetchCollections(projectId);
    },
    // 保存集合级变量（EnvVar[] 序列化为 JSON 写入 variables 列）。
    async saveCollectionVariables(
      projectId: number,
      collectionId: number,
      vars: { key: string; value: string; enabled: boolean; secret: boolean }[],
    ) {
      const c = this.collections.find((x) => x.id === collectionId);
      if (c) c.variables = JSON.stringify(vars);
      await updateCollection(projectId, collectionId, { variables: JSON.stringify(vars) });
    },
    async fetchHistory(projectId: number, requestId: number) {
      this.histories[requestId] = await listHistory(projectId, requestId);
      return this.histories[requestId];
    },
    reset() {
      this.collections = [];
      this.requestsByCollection = {};
      this.histories = {};
    },
  },
  getters: {
    // 某集合及其所有父文件夹的变量汇总（父 → 子覆盖），返回扁平字典。
    mergedVarsOf(state) {
      return (collectionId: number): Record<string, string> => {
        const chain: Collection[] = [];
        let cur = state.collections.find((c) => c.id === collectionId);
        while (cur) {
          chain.unshift(cur);
          cur = cur.parentId
            ? state.collections.find((c) => c.id === cur!.parentId)
            : undefined;
        }
        const out: Record<string, string> = {};
        for (const c of chain) {
          if (!c.variables) continue;
          try {
            const arr = JSON.parse(c.variables) as { key: string; value: string }[];
            for (const v of arr) if (v.key && v.key.trim()) out[v.key.trim()] = v.value;
          } catch {
            /* ignore malformed */
          }
        }
        return out;
      };
    },
  },
});
