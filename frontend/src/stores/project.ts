import { defineStore } from "pinia";
import { listProjects, listMembers } from "@/api/project";
import {
  createProject,
  deleteProject,
  updateProject,
  addMember,
  removeMember,
  updateMember,
} from "@/api/project";
import type { Project, ProjectMember } from "@/types/project";

// 项目与成员状态。
export const useProjectStore = defineStore("project", {
  state: () => ({
    projects: [] as Project[],
    current: null as Project | null,
    members: [] as ProjectMember[],
    total: 0,
    page: 1,
    perPage: 200,
  }),
  actions: {
    async fetchProjects(page = 1, perPage = 200) {
      const p = await listProjects(page, perPage);
      this.projects = p.items;
      this.total = p.total;
      this.page = p.page;
      this.perPage = p.perPage;
    },
    setCurrent(p: Project) {
      this.current = p;
    },
    async create(name: string, description: string) {
      const p = await createProject(name, description);
      this.projects.unshift(p);
      return p;
    },
    async update(id: number, name: string, description: string) {
      await updateProject(id, name, description);
      await this.fetchProjects();
    },
    async remove(id: number) {
      await deleteProject(id);
      this.projects = this.projects.filter((p) => p.id !== id);
    },
    async fetchMembers(projectId: number) {
      this.members = await listMembers(projectId);
    },
    async addMember(
      projectId: number,
      userId: number,
      role: string,
      permissions: Record<string, boolean>
    ) {
      await addMember(projectId, userId, role, permissions);
      await this.fetchMembers(projectId);
    },
    async removeMember(projectId: number, userId: number) {
      await removeMember(projectId, userId);
      await this.fetchMembers(projectId);
    },
    async updateMember(
      projectId: number,
      userId: number,
      role: string,
      permissions: Record<string, boolean>
    ) {
      await updateMember(projectId, userId, role, permissions);
      await this.fetchMembers(projectId);
    },
  },
});
